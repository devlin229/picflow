package service

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"picflow/backend/internal/errno"
	"picflow/backend/internal/imageproc"
	"picflow/backend/internal/model"
	"picflow/backend/internal/storage"
	"picflow/backend/internal/svc"
	"picflow/backend/internal/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxTaskAssets = 100

// TaskService 处理上传、任务查询、尺寸图和下载业务。
type TaskService struct {
	svcCtx *svc.ServiceContext
}

func NewTaskService(svcCtx *svc.ServiceContext) *TaskService {
	return &TaskService{svcCtx: svcCtx}
}

func (s *TaskService) Create(files []*multipart.FileHeader, request types.ProcessConfigRequest) (*types.TaskResponse, error) {
	if len(files) == 0 {
		return nil, errno.InvalidArgument("请至少上传一张图片")
	}
	if len(files) > maxTaskAssets {
		return nil, errno.InvalidArgument("单个任务最多上传 100 张图片")
	}
	config, err := s.resolveProcessConfig(request)
	if err != nil {
		return nil, err
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, errno.Internal(err)
	}
	task := &model.Task{
		ID: uuid.NewString(), Type: "standardize", Status: model.TaskStatusQueued, ConfigJSON: string(configJSON),
		TotalAssets: len(files), CompletedAssets: 0,
	}
	defer func() {
		if err != nil {
			_ = s.svcCtx.Storage.RemoveTask(task.ID)
		}
	}()
	for _, header := range files {
		asset, saveErr := s.saveAsset(task.ID, header)
		if saveErr != nil {
			err = saveErr
			return nil, err
		}
		task.Assets = append(task.Assets, *asset)
	}
	if err = s.svcCtx.Tasks.Create(task); err != nil {
		return nil, errno.Internal(fmt.Errorf("保存任务失败: %w", err))
	}
	if err = s.svcCtx.Workers.Enqueue(task.ID); err != nil {
		_ = s.svcCtx.Tasks.Delete(task.ID)
		return nil, errno.Unavailable(err)
	}
	return s.Get(task.ID)
}

func (s *TaskService) Get(taskID string) (*types.TaskResponse, error) {
	task, err := s.svcCtx.Tasks.Get(taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.NotFound("处理任务不存在")
	}
	if err != nil {
		return nil, errno.Internal(err)
	}
	return taskResponse(task), nil
}

func (s *TaskService) CreateSizeChart(taskID string, request types.SizeChartRequest) (*types.TaskOutputResponse, error) {
	if _, err := s.Get(taskID); err != nil {
		return nil, err
	}
	template, err := s.svcCtx.Templates.Get(request.TemplateID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && template.Type != "size_chart") {
		return nil, errno.InvalidArgument("尺寸图模板不存在")
	}
	if err != nil {
		return nil, errno.Internal(err)
	}
	asset, err := s.svcCtx.Tasks.GetAsset(taskID, request.AssetID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.NotFound("任务中的原图不存在")
	}
	if err != nil {
		return nil, errno.Internal(err)
	}
	outputID := uuid.NewString()
	templateConfig, err := imageproc.NormalizeSizeTemplateConfig([]byte(template.ConfigJSON))
	if err != nil {
		return nil, errno.Internal(fmt.Errorf("尺寸图模板配置无效: %w", err))
	}
	path, err := s.svcCtx.Storage.OutputPath(taskID, outputID, templateConfig.OutputFormat)
	if err != nil {
		return nil, errno.Internal(err)
	}
	if err = imageproc.SizeChart(asset.SourcePath, path, imageproc.SizeConfig{
		Width: request.Width, Height: request.Height, Depth: request.Depth, Unit: request.Unit,
	}, templateConfig); err != nil {
		return nil, errno.Internal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		_ = os.Remove(path)
		return nil, errno.Internal(err)
	}
	output := &model.Output{
		ID: outputID, TaskID: taskID, AssetID: asset.ID, Type: model.OutputTypeSizeChart,
		Filename: outputFilename(asset.Filename, "size", templateConfig.OutputFormat), OutputPath: path,
		Width: templateConfig.CanvasWidth, Height: templateConfig.CanvasHeight, Format: templateConfig.OutputFormat, Size: info.Size(),
	}
	annotation := &model.SizeAnnotation{
		ID: uuid.NewString(), TaskID: taskID, AssetID: asset.ID, OutputID: outputID,
		Width: request.Width, Height: request.Height, Depth: request.Depth, Unit: request.Unit, TemplateID: request.TemplateID,
	}
	if err = s.svcCtx.Tasks.AddSizeResult(output, annotation); err != nil {
		_ = os.Remove(path)
		return nil, errno.Internal(err)
	}
	value := outputResponse(*output)
	return &value, nil
}

func (s *TaskService) Output(outputID string) (*model.Output, error) {
	output, err := s.svcCtx.Tasks.GetOutput(outputID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.NotFound("输出文件不存在")
	}
	if err != nil {
		return nil, errno.Internal(err)
	}
	if _, err = os.Stat(output.OutputPath); errors.Is(err, os.ErrNotExist) {
		return nil, errno.NotFound("输出文件已不存在")
	}
	if err != nil {
		return nil, errno.Internal(err)
	}
	return output, nil
}

// CreateArchive 将任务输出写入临时 ZIP 文件，调用方使用完后必须删除该文件。
func (s *TaskService) CreateArchive(taskID string) (string, error) {
	task, err := s.svcCtx.Tasks.Get(taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errno.NotFound("处理任务不存在")
	}
	if err != nil {
		return "", errno.Internal(err)
	}
	if len(task.Outputs) == 0 {
		return "", errno.InvalidArgument("当前任务还没有可下载的结果")
	}
	temp, err := os.CreateTemp(s.svcCtx.Config.DataDir, "picflow-*.zip")
	if err != nil {
		return "", errno.Internal(err)
	}
	path := temp.Name()
	archive := zip.NewWriter(temp)
	names := make(map[string]int)
	for _, output := range task.Outputs {
		entryName := output.Filename
		if names[entryName] > 0 {
			entryName = strings.TrimSuffix(entryName, filepath.Ext(entryName)) + "_" + output.ID[:8] + filepath.Ext(entryName)
		}
		names[output.Filename]++
		if err = addZipFile(archive, entryName, output.OutputPath); err != nil {
			_ = archive.Close()
			_ = temp.Close()
			_ = os.Remove(path)
			return "", errno.Internal(err)
		}
	}
	if err = archive.Close(); err != nil {
		_ = temp.Close()
		_ = os.Remove(path)
		return "", errno.Internal(err)
	}
	if err = temp.Close(); err != nil {
		_ = os.Remove(path)
		return "", errno.Internal(err)
	}
	return path, nil
}

func (s *TaskService) Delete(taskID string) error {
	task, err := s.svcCtx.Tasks.Get(taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errno.NotFound("处理任务不存在")
	}
	if err != nil {
		return errno.Internal(err)
	}
	if task.Status == model.TaskStatusQueued || task.Status == model.TaskStatusProcessing {
		return errno.Conflict("任务正在处理，暂时不能删除")
	}
	if err := s.svcCtx.Tasks.Delete(taskID); err != nil {
		return errno.Internal(err)
	}
	if err := s.svcCtx.Storage.RemoveTask(taskID); err != nil {
		return errno.Internal(err)
	}
	return nil
}

func (s *TaskService) DeleteAsset(taskID, assetID string) error {
	task, err := s.svcCtx.Tasks.Get(taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errno.NotFound("处理任务不存在")
	}
	if err != nil {
		return errno.Internal(err)
	}
	if task.Status == model.TaskStatusQueued || task.Status == model.TaskStatusProcessing {
		return errno.Conflict("任务正在处理，暂时不能删除图片")
	}
	asset, outputs, err := s.svcCtx.Tasks.DeleteAsset(taskID, assetID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errno.NotFound("任务中的原图不存在")
	}
	if err != nil {
		return errno.Internal(err)
	}
	_ = os.Remove(asset.SourcePath)
	for _, output := range outputs {
		_ = os.Remove(output.OutputPath)
	}
	return nil
}

func (s *TaskService) saveAsset(taskID string, header *multipart.FileHeader) (*model.Asset, error) {
	if header.Size > s.svcCtx.Config.MaxUploadSize {
		return nil, errno.InvalidArgument(fmt.Sprintf("图片 %s 超过大小限制", filepath.Base(header.Filename)))
	}
	file, err := header.Open()
	if err != nil {
		return nil, errno.InvalidArgument("读取上传图片失败")
	}
	defer file.Close()
	assetID := uuid.NewString()
	tempPath, size, err := s.svcCtx.Storage.SaveUpload(taskID, assetID, file, s.svcCtx.Config.MaxUploadSize)
	if err != nil {
		return nil, errno.InvalidArgument(err.Error())
	}
	width, height, format, err := imageproc.Inspect(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		return nil, errno.InvalidArgument(fmt.Sprintf("图片 %s 无效: %v", filepath.Base(header.Filename), err))
	}
	if width < 1 || height < 1 || int64(width)*int64(height) > 50_000_000 {
		_ = os.Remove(tempPath)
		return nil, errno.InvalidArgument(fmt.Sprintf("图片 %s 的分辨率不受支持", filepath.Base(header.Filename)))
	}
	path, err := s.svcCtx.Storage.FinalizeUpload(tempPath, format)
	if err != nil {
		_ = os.Remove(tempPath)
		return nil, errno.Internal(err)
	}
	return &model.Asset{
		ID: assetID, TaskID: taskID, Filename: filepath.Base(header.Filename), SourcePath: path,
		Width: width, Height: height, Format: format, Size: size,
	}, nil
}

func validateProcessConfig(request types.ProcessConfigRequest) (imageproc.ProcessConfig, error) {
	if request.CanvasWidth < 64 || request.CanvasWidth > 4096 || request.CanvasHeight < 64 || request.CanvasHeight > 4096 {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("画布宽高必须在 64 到 4096 像素之间")
	}
	marginMode := request.MarginMode
	if marginMode == "" {
		marginMode = "fixed"
	}
	if marginMode != "fixed" && marginMode != "auto" {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("边距模式仅支持 fixed 或 auto")
	}
	margin := request.Margin
	if marginMode == "auto" {
		margin = min(request.CanvasWidth, request.CanvasHeight) * 8 / 100
	}
	if margin < 0 || margin*2 >= request.CanvasWidth || margin*2 >= request.CanvasHeight {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("边距不能超过画布范围")
	}
	if request.LayoutMode != "center_fit" || !request.KeepSubjectComplete {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("V1 仅支持居中、等比缩放和保持主体完整")
	}
	if storage.Extension(request.OutputFormat) == "" {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("输出格式仅支持 jpeg、png 和 webp")
	}
	if request.Background == "transparent" && (request.OutputFormat == "jpeg" || request.OutputFormat == "jpg") {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("JPEG 不支持透明背景")
	}
	config := imageproc.ProcessConfig{
		CanvasWidth: request.CanvasWidth, CanvasHeight: request.CanvasHeight, Background: request.Background,
		LayoutMode: request.LayoutMode, KeepSubjectComplete: request.KeepSubjectComplete,
		OutputFormat: request.OutputFormat, MarginMode: marginMode, Margin: margin,
	}
	// 通过处理器的背景色校验，避免任务进入队列后才失败。
	if request.Background != "transparent" {
		value := strings.TrimPrefix(request.Background, "#")
		if len(value) != 6 {
			return imageproc.ProcessConfig{}, errno.InvalidArgument("背景色必须是 transparent 或 #RRGGBB")
		}
		for _, char := range value {
			if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
				return imageproc.ProcessConfig{}, errno.InvalidArgument("背景色格式无效")
			}
		}
	}
	return config, nil
}

func (s *TaskService) resolveProcessConfig(request types.ProcessConfigRequest) (imageproc.ProcessConfig, error) {
	if strings.TrimSpace(request.TemplateID) == "" {
		return validateProcessConfig(request)
	}
	template, err := s.svcCtx.Templates.Get(request.TemplateID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && template.Type != "main_image") {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("主图模板不存在")
	}
	if err != nil {
		return imageproc.ProcessConfig{}, errno.Internal(err)
	}
	var templateRequest types.ProcessConfigRequest
	if err = json.Unmarshal([]byte(template.ConfigJSON), &templateRequest); err != nil {
		return imageproc.ProcessConfig{}, errno.Internal(fmt.Errorf("解析主图模板失败: %w", err))
	}
	templateRequest.TemplateID = template.ID
	return validateProcessConfig(templateRequest)
}

func taskResponse(task *model.Task) *types.TaskResponse {
	result := &types.TaskResponse{
		ID: task.ID, Type: task.Type, Status: task.Status, ErrorMessage: task.ErrorMessage,
		TotalAssets: task.TotalAssets, CompletedAssets: task.CompletedAssets,
		Assets: make([]types.TaskAssetResponse, 0, len(task.Assets)), Outputs: make([]types.TaskOutputResponse, 0, len(task.Outputs)),
		CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
	}
	for _, asset := range task.Assets {
		result.Assets = append(result.Assets, types.TaskAssetResponse{
			ID: asset.ID, TaskID: asset.TaskID, Filename: asset.Filename, Width: asset.Width,
			Height: asset.Height, Format: asset.Format, Size: asset.Size,
		})
	}
	for _, output := range task.Outputs {
		result.Outputs = append(result.Outputs, outputResponse(output))
	}
	return result
}

func outputResponse(output model.Output) types.TaskOutputResponse {
	return types.TaskOutputResponse{
		ID: output.ID, TaskID: output.TaskID, AssetID: output.AssetID, Type: output.Type,
		Filename: output.Filename, Width: output.Width, Height: output.Height, Format: output.Format,
		Size: output.Size, DownloadURL: "/api/outputs/" + output.ID + "/download",
	}
}

func outputFilename(original, suffix, format string) string {
	name := strings.TrimSuffix(filepath.Base(original), filepath.Ext(original))
	if strings.TrimSpace(name) == "" {
		name = "image"
	}
	return name + "_" + suffix + storage.Extension(format)
}

func addZipFile(archive *zip.Writer, name, path string) error {
	source, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开输出文件失败: %w", err)
	}
	defer source.Close()
	entry, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("创建 ZIP 条目失败: %w", err)
	}
	if _, err = io.Copy(entry, source); err != nil {
		return fmt.Errorf("写入 ZIP 失败: %w", err)
	}
	return nil
}
