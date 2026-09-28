package types

import "time"

// ProcessConfigRequest 是前端提交的标准化处理配置。
type ProcessConfigRequest struct {
	TemplateID          string `json:"template_id"`
	CanvasWidth         int    `json:"canvas_width"`
	CanvasHeight        int    `json:"canvas_height"`
	Background          string `json:"background"`
	LayoutMode          string `json:"layout_mode"`
	KeepSubjectComplete bool   `json:"keep_subject_complete"`
	OutputFormat        string `json:"output_format"`
	MarginMode          string `json:"margin_mode"`
	Margin              int    `json:"margin"`
}

type SizeChartRequest struct {
	AssetID    string  `json:"asset_id" binding:"required"`
	Width      float64 `json:"width" binding:"required,gt=0"`
	Height     float64 `json:"height" binding:"required,gt=0"`
	Depth      float64 `json:"depth" binding:"gte=0"`
	Unit       string  `json:"unit" binding:"required,oneof=cm mm"`
	TemplateID string  `json:"template_id" binding:"required"`
}

type TaskAssetResponse struct {
	ID       string `json:"id"`
	TaskID   string `json:"task_id"`
	Filename string `json:"filename"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Size     int64  `json:"size"`
}

type TaskOutputResponse struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	AssetID     string `json:"asset_id"`
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Format      string `json:"format"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

type TaskResponse struct {
	ID              string               `json:"id"`
	Type            string               `json:"type"`
	Status          string               `json:"status"`
	ErrorMessage    string               `json:"error_message,omitempty"`
	TotalAssets     int                  `json:"total_assets"`
	CompletedAssets int                  `json:"completed_assets"`
	Assets          []TaskAssetResponse  `json:"assets"`
	Outputs         []TaskOutputResponse `json:"outputs"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}
