package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"picflow/backend/internal/imageproc"
	"picflow/backend/internal/llm"
	"picflow/backend/internal/model"
	"picflow/backend/internal/repository"
	"picflow/backend/internal/storage"

	"github.com/google/uuid"
)

// Pool 使用固定数量的 Worker 异步执行规则型图片处理任务。
type Pool struct {
	queue   chan string
	workers int
	tasks   *repository.TaskRepository
	storage *storage.Storage
	logger  *slog.Logger
	wg      sync.WaitGroup
	editor  llm.ImageEditor
}

func New(workers int, tasks *repository.TaskRepository, files *storage.Storage, logger *slog.Logger, editor llm.ImageEditor) *Pool {
	return &Pool{queue: make(chan string, 100), workers: workers, tasks: tasks, storage: files, logger: logger, editor: editor}
}

func (p *Pool) Start() {
	for range p.workers {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for taskID := range p.queue {
				p.processSafely(taskID)
			}
		}()
	}
}

func (p *Pool) Enqueue(taskID string) error {
	select {
	case p.queue <- taskID:
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

func (p *Pool) Close() {
	close(p.queue)
	p.wg.Wait()
}

func (p *Pool) processSafely(taskID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			p.fail(taskID, fmt.Errorf("处理任务发生异常: %v", recovered))
		}
	}()
	p.process(taskID)
}

func (p *Pool) process(taskID string) {
	task, err := p.tasks.Get(taskID)
	if err != nil {
		p.logger.Error("读取处理任务失败", slog.String("task_id", taskID), slog.Any("error", err))
		return
	}
	if err = p.tasks.UpdateStatus(taskID, model.TaskStatusProcessing, ""); err != nil {
		p.logger.Error("更新任务状态失败", slog.String("task_id", taskID), slog.Any("error", err))
		return
	}
	var config imageproc.ProcessConfig
	if err = json.Unmarshal([]byte(task.ConfigJSON), &config); err != nil {
		p.fail(taskID, fmt.Errorf("解析处理配置失败: %w", err))
		return
	}
	for _, asset := range task.Assets {
		outputID := uuid.NewString()
		path, pathErr := p.storage.OutputPath(taskID, outputID, config.OutputFormat)
		if pathErr != nil {
			p.fail(taskID, pathErr)
			return
		}
		if err = p.processAsset(asset, path, config); err != nil {
			_ = os.Remove(path)
			p.fail(taskID, fmt.Errorf("处理图片 %s 失败: %w", asset.Filename, err))
			return
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			_ = os.Remove(path)
			p.fail(taskID, fmt.Errorf("读取输出文件失败: %w", statErr))
			return
		}
		output := &model.Output{
			ID: outputID, TaskID: taskID, AssetID: asset.ID, Type: model.OutputTypeStandardized,
			Filename: outputFilename(asset.Filename, "processed", config.OutputFormat), OutputPath: path,
			Width: config.CanvasWidth, Height: config.CanvasHeight, Format: config.OutputFormat, Size: info.Size(),
		}
		if err = p.tasks.AddOutput(output); err != nil {
			_ = os.Remove(path)
			p.fail(taskID, fmt.Errorf("保存输出记录失败: %w", err))
			return
		}
		if err = p.tasks.UpdateProgress(taskID, task.CompletedAssets+1); err != nil {
			p.fail(taskID, fmt.Errorf("更新任务进度失败: %w", err))
			return
		}
		task.CompletedAssets++
	}
	if err = p.tasks.UpdateStatus(taskID, model.TaskStatusSucceeded, ""); err != nil {
		p.logger.Error("更新任务完成状态失败", slog.String("task_id", taskID), slog.Any("error", err))
	}
}

func (p *Pool) processAsset(asset model.Asset, outputPath string, config imageproc.ProcessConfig) error {
	if !config.AIBackground {
		return imageproc.Standardize(asset.SourcePath, outputPath, config)
	}
	if p.editor == nil {
		return fmt.Errorf("AI 换背景尚未配置")
	}
	config.Background, config.AIBackgroundPrompt = imageproc.ResolveAIBackground(config.Background, config.AIBackgroundPrompt)
	path, err := p.storage.AIResultPath(asset.TaskID, asset.ID)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		p.logger.Info("复用已保存的 AI 图片，不再次请求模型", slog.String("task_id", asset.TaskID), slog.String("asset_id", asset.ID))
		return p.standardizeAIResult(path, outputPath, config)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("读取已保存 AI 图片失败")
	}
	file, err := os.Open(asset.SourcePath)
	if err != nil {
		return fmt.Errorf("读取 AI 原图失败")
	}
	data, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	_ = file.Close()
	if err != nil || len(data) > 10<<20 {
		return fmt.Errorf("读取 AI 原图失败或图片超过 10MB")
	}
	background := "将图1中商品以外的全部背景区域（包括四周和空隙）替换为均匀的纯色 " + config.Background + "。不得保留原来的背景色。不要渐变、阴影或场景元素。商品自身的白色或其他颜色必须保留。"
	if config.AIBackgroundPrompt != "" {
		background = "将图1的原背景替换为以下场景：" + config.AIBackgroundPrompt
	}
	prompt := background + " 编辑图1，不是复制原图。只改变背景，商品主体、形状、材质、颜色及配件保持不变；商品完整呈现，不裁切、不变形。不要添加新的文字、边框或水印。"
	p.logger.Info("开始 AI 换背景", slog.String("task_id", asset.TaskID), slog.String("asset_id", asset.ID), slog.String("background", config.Background), slog.Bool("scene_background", config.AIBackgroundPrompt != ""))
	result, err := p.editor.Edit(context.Background(), llm.ImageInput{Data: data, MIME: "image/" + asset.Format, Prompt: prompt})
	if err != nil {
		return err
	}
	path, err = p.storage.SaveAIResult(asset.TaskID, asset.ID, result)
	if err != nil {
		return err
	}
	p.logger.Info("AI 原始返回图已保存", slog.String("task_id", asset.TaskID), slog.String("asset_id", asset.ID), slog.String("path", path))
	return p.standardizeAIResult(path, outputPath, config)
}

func (p *Pool) standardizeAIResult(path, outputPath string, config imageproc.ProcessConfig) error {
	width, height, _, err := imageproc.Inspect(path)
	if err != nil || int64(width)*int64(height) > 50_000_000 {
		return fmt.Errorf("AI 返回的图片格式或分辨率无效")
	}
	if config.AIBackgroundPrompt == "" {
		if err = imageproc.ValidateAIBackground(path, config.Background); err != nil {
			return err
		}
	}
	config.ReplaceSimpleBackground = false
	return imageproc.Standardize(path, outputPath, config)
}

func (p *Pool) fail(taskID string, err error) {
	p.logger.Error("图片处理任务失败", slog.String("task_id", taskID), slog.Any("error", err))
	outputs, cleanupErr := p.tasks.DeleteStandardizedOutputs(taskID)
	if cleanupErr != nil {
		p.logger.Error("清理失败任务输出记录失败", slog.String("task_id", taskID), slog.Any("error", cleanupErr))
	} else {
		for _, output := range outputs {
			if removeErr := os.Remove(output.OutputPath); removeErr != nil && !os.IsNotExist(removeErr) {
				p.logger.Error("清理失败任务输出文件失败", slog.String("task_id", taskID), slog.String("path", output.OutputPath), slog.Any("error", removeErr))
			}
		}
	}
	if progressErr := p.tasks.UpdateProgress(taskID, 0); progressErr != nil {
		p.logger.Error("重置失败任务进度失败", slog.String("task_id", taskID), slog.Any("error", progressErr))
	}
	if updateErr := p.tasks.UpdateStatus(taskID, model.TaskStatusFailed, err.Error()); updateErr != nil {
		p.logger.Error("更新任务失败状态失败", slog.String("task_id", taskID), slog.Any("error", updateErr))
	}
}

func outputFilename(original, suffix, format string) string {
	name := strings.TrimSuffix(filepath.Base(original), filepath.Ext(original))
	if strings.TrimSpace(name) == "" {
		name = "image"
	}
	return name + "_" + suffix + storage.Extension(format)
}
