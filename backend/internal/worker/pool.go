package worker

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"picflow/backend/internal/imageproc"
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
}

func New(workers int, tasks *repository.TaskRepository, files *storage.Storage, logger *slog.Logger) *Pool {
	return &Pool{queue: make(chan string, 100), workers: workers, tasks: tasks, storage: files, logger: logger}
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
		if err = imageproc.Standardize(asset.SourcePath, path, config); err != nil {
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
