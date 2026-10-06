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
	"strconv"
	"strings"
	"unicode/utf8"

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

// TaskService 处理上传、任务查询、规格图和下载业务。
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
	if config.AIBackground && s.svcCtx.ImageEditor == nil {
		return nil, errno.Unavailable(fmt.Errorf("AI 图片处理尚未配置，请在后端设置 LLM_API_KEY"))
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
		if config.AIBackground && asset.Size > 10<<20 {
			err = errno.InvalidArgument("AI 换背景单张原图不能超过 10MB")
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

// List 查询已保存的历史任务，不重新处理图片或调用模型。
func (s *TaskService) List(input types.TaskListRequest) (*types.TaskListResponse, error) {
	if input.Page == 0 {
		input.Page = 1
	}
	if input.PageSize == 0 {
		input.PageSize = 12
	}
	if input.Page < 1 || input.Page > 1000000 || input.PageSize < 1 || input.PageSize > 50 {
		return nil, errno.InvalidArgument("页码须为 1 至 1000000，每页数量须为 1 至 50")
	}
	tasks, total, page, err := s.svcCtx.Tasks.List(input.Page, input.PageSize)
	if err != nil {
		return nil, errno.Internal(err)
	}
	items := make([]*types.TaskResponse, 0, len(tasks))
	for i := range tasks {
		items = append(items, taskResponse(&tasks[i]))
	}
	return &types.TaskListResponse{Items: items, Total: total, Page: page, PageSize: input.PageSize}, nil
}

// Retry 重新把失败任务放入处理队列，不重复上传原图。
func (s *TaskService) Retry(taskID string) (*types.TaskResponse, error) {
	task, err := s.svcCtx.Tasks.Get(taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errno.NotFound("处理任务不存在")
	}
	if err != nil {
		return nil, errno.Internal(err)
	}
	if task.Status != model.TaskStatusFailed {
		return nil, errno.Conflict("只有失败任务可以重试")
	}
	outputs, err := s.svcCtx.Tasks.DeleteStandardizedOutputs(taskID)
	if err != nil {
		return nil, errno.Internal(err)
	}
	for _, output := range outputs {
		_ = os.Remove(output.OutputPath)
	}
	if err = s.svcCtx.Tasks.ResetForRetry(taskID); err != nil {
		return nil, errno.Internal(err)
	}
	if err = s.svcCtx.Workers.Enqueue(taskID); err != nil {
		_ = s.svcCtx.Tasks.UpdateStatus(taskID, model.TaskStatusFailed, "任务重新入队失败，请稍后再试")
		return nil, errno.Unavailable(err)
	}
	return s.Get(taskID)
}

func (s *TaskService) CreateSizeChart(taskID string, request types.SizeChartRequest) (*types.TaskOutputResponse, error) {
	if _, err := s.Get(taskID); err != nil {
		return nil, err
	}
	template, err := s.svcCtx.Templates.Get(request.TemplateID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && template.Type != "size_chart") {
		return nil, errno.InvalidArgument("规格图模板不存在")
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
		return nil, errno.Internal(fmt.Errorf("规格图模板配置无效: %w", err))
	}
	specifications, err := normalizeSpecifications(request.Specifications)
	if err != nil {
		return nil, err
	}
	if err = validateTablePosition(request.TablePosition); err != nil {
		return nil, err
	}
	style, err := normalizeSpecificationStyle(request.TableStyle)
	if err != nil {
		return nil, err
	}
	path, err := s.svcCtx.Storage.OutputPath(taskID, outputID, templateConfig.OutputFormat)
	if err != nil {
		return nil, errno.Internal(err)
	}
	if err = imageproc.SpecificationChart(asset.SourcePath, path, imageproc.SpecificationConfig{
		Items: specifications, TablePreset: request.TablePosition.Preset,
		TableX: request.TablePosition.X, TableY: request.TablePosition.Y, TableWidth: request.TablePosition.Width,
		Style: style,
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
		Filename: outputFilename(asset.Filename, "spec", templateConfig.OutputFormat), OutputPath: path,
		Width: templateConfig.CanvasWidth, Height: templateConfig.CanvasHeight, Format: templateConfig.OutputFormat, Size: info.Size(),
	}
	specificationsJSON, err := json.Marshal(specifications)
	if err != nil {
		_ = os.Remove(path)
		return nil, errno.Internal(err)
	}
	styleJSON, err := json.Marshal(style)
	if err != nil {
		_ = os.Remove(path)
		return nil, errno.Internal(err)
	}
	annotation := &model.SizeAnnotation{
		ID: uuid.NewString(), TaskID: taskID, AssetID: asset.ID, OutputID: outputID,
		SpecificationsJSON: string(specificationsJSON), StyleJSON: string(styleJSON), TablePreset: request.TablePosition.Preset,
		TableX: request.TablePosition.X, TableY: request.TablePosition.Y, TableWidth: request.TablePosition.Width, TemplateID: request.TemplateID,
	}
	if err = s.svcCtx.Tasks.AddSizeResult(output, annotation); err != nil {
		_ = os.Remove(path)
		return nil, errno.Internal(err)
	}
	value := outputResponse(*output)
	return &value, nil
}

func normalizeSpecificationStyle(value types.SpecificationTableStyleRequest) (imageproc.SpecificationStyle, error) {
	if value.Style != "simple-table" {
		return imageproc.SpecificationStyle{}, errno.InvalidArgument("规格表格样式无效")
	}
	if value.BorderWidth < 1 || value.BorderWidth > 8 {
		return imageproc.SpecificationStyle{}, errno.InvalidArgument("规格表格边框粗细必须在 1 到 8 像素之间")
	}
	if !isHexColor(value.BorderColor) {
		return imageproc.SpecificationStyle{}, errno.InvalidArgument("规格表格边框色必须是 #RRGGBB")
	}
	if value.BackgroundColor != "transparent" && !isHexColor(value.BackgroundColor) {
		return imageproc.SpecificationStyle{}, errno.InvalidArgument("规格表格背景色必须是 transparent 或 #RRGGBB")
	}
	if !isHexColor(value.TextColor) {
		return imageproc.SpecificationStyle{}, errno.InvalidArgument("规格表格文字色必须是 #RRGGBB")
	}
	return imageproc.SpecificationStyle{
		Style: value.Style, BorderWidth: value.BorderWidth, BorderColor: value.BorderColor,
		BackgroundColor: value.BackgroundColor, TextColor: value.TextColor,
	}, nil
}

func isHexColor(value string) bool {
	hexValue := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(hexValue) != 6 {
		return false
	}
	_, err := strconv.ParseUint(hexValue, 16, 32)
	return err == nil
}

func validateTablePosition(value types.SpecificationTablePositionRequest) error {
	switch value.Preset {
	case "top-left", "top-right", "bottom-left", "bottom-right", "custom":
	default:
		return errno.InvalidArgument("规格表格位置预设无效")
	}
	if value.X < 0 || value.X > 1 || value.Y < 0 || value.Y > 1 {
		return errno.InvalidArgument("规格表格位置必须在图片范围内")
	}
	if value.Width < 0.24 || value.Width > 0.8 {
		return errno.InvalidArgument("规格表格宽度必须在画布宽度的 24% 到 80% 之间")
	}
	return nil
}

func normalizeTaskSpecification(value *types.TaskSpecificationRequest) (*imageproc.SpecificationConfig, error) {
	if value == nil {
		return nil, nil
	}
	items, err := normalizeSpecifications(value.Specifications)
	if err != nil {
		return nil, err
	}
	if err = validateTablePosition(value.TablePosition); err != nil {
		return nil, err
	}
	style, err := normalizeSpecificationStyle(value.TableStyle)
	if err != nil {
		return nil, err
	}
	return &imageproc.SpecificationConfig{
		Items: items, TablePreset: value.TablePosition.Preset,
		TableX: value.TablePosition.X, TableY: value.TablePosition.Y, TableWidth: value.TablePosition.Width,
		Style: style,
	}, nil
}

func normalizeSpecifications(values []types.SpecificationRequest) ([]imageproc.Specification, error) {
	if len(values) == 0 || len(values) > 10 {
		return nil, errno.InvalidArgument("规格参数数量必须在 1 到 10 项之间")
	}
	result := make([]imageproc.Specification, 0, len(values))
	for _, value := range values {
		label := strings.TrimSpace(value.Label)
		text := strings.TrimSpace(value.Value)
		if label == "" || text == "" {
			return nil, errno.InvalidArgument("规格名称和值不能为空")
		}
		if utf8.RuneCountInString(label) > 20 || utf8.RuneCountInString(text) > 40 {
			return nil, errno.InvalidArgument("规格名称最多 20 个字符，规格值最多 40 个字符")
		}
		result = append(result, imageproc.Specification{Label: label, Value: text})
	}
	return result, nil
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
	layoutMode := request.LayoutMode
	if layoutMode == "" || layoutMode == "center_fit" {
		layoutMode = "contain"
	}
	switch layoutMode {
	case "contain", "cover-center", "cover-top", "cover-bottom":
	default:
		return imageproc.ProcessConfig{}, errno.InvalidArgument("缩放裁剪模式无效")
	}
	margin := request.Margin
	if marginMode == "auto" {
		margin = min(request.CanvasWidth, request.CanvasHeight) * 8 / 100
	}
	if margin < 0 || (layoutMode == "contain" && (margin*2 >= request.CanvasWidth || margin*2 >= request.CanvasHeight)) {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("边距不能超过画布范围")
	}
	if storage.Extension(request.OutputFormat) == "" {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("输出格式仅支持 jpeg、png 和 webp")
	}
	if request.Background == "transparent" && (request.OutputFormat == "jpeg" || request.OutputFormat == "jpg") {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("JPEG 不支持透明背景")
	}
	quality := request.OutputQuality
	if quality == 0 {
		quality = 90
	}
	if quality < 30 || quality > 100 {
		return imageproc.ProcessConfig{}, errno.InvalidArgument("输出质量必须在 30 到 100 之间")
	}
	config := imageproc.ProcessConfig{
		CanvasWidth: request.CanvasWidth, CanvasHeight: request.CanvasHeight, Background: request.Background,
		LayoutMode: layoutMode, KeepSubjectComplete: layoutMode == "contain",
		OutputFormat: request.OutputFormat, MarginMode: marginMode, Margin: margin,
		OutputQuality: quality,
		AIBackground:  strings.TrimSpace(request.AIBackgroundPrompt) != "", AIBackgroundPrompt: strings.TrimSpace(request.AIBackgroundPrompt),
	}
	if config.AIBackground {
		if request.Background == "transparent" {
			return imageproc.ProcessConfig{}, errno.InvalidArgument("当前 AI 图片处理不支持透明底")
		}
		if utf8.RuneCountInString(config.AIBackgroundPrompt) > 1000 {
			return imageproc.ProcessConfig{}, errno.InvalidArgument("AI 提示词不能超过 1000 字")
		}
	}
	specification, err := normalizeTaskSpecification(request.Specification)
	if err != nil {
		return imageproc.ProcessConfig{}, err
	}
	config.Specification = specification
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
	templateRequest.Specification = request.Specification
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
