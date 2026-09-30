package types

import "time"

// ProcessConfigRequest 是前端提交的标准化处理配置。
type ProcessConfigRequest struct {
	TemplateID              string                    `json:"template_id"`
	CanvasWidth             int                       `json:"canvas_width"`
	CanvasHeight            int                       `json:"canvas_height"`
	Background              string                    `json:"background"`
	LayoutMode              string                    `json:"layout_mode"`
	KeepSubjectComplete     bool                      `json:"keep_subject_complete"`
	OutputFormat            string                    `json:"output_format"`
	MarginMode              string                    `json:"margin_mode"`
	Margin                  int                       `json:"margin"`
	OutputQuality           int                       `json:"output_quality"`
	ReplaceSimpleBackground bool                      `json:"replace_simple_background"`
	BackgroundTolerance     int                       `json:"background_tolerance"`
	Specification           *TaskSpecificationRequest `json:"specification,omitempty"`
}

type SpecificationRequest struct {
	Label string `json:"label" binding:"required,max=20"`
	Value string `json:"value" binding:"required,max=40"`
}

type SpecificationTablePositionRequest struct {
	Preset string  `json:"preset" binding:"required,oneof=top-left top-right bottom-left bottom-right custom"`
	X      float64 `json:"x" binding:"gte=0,lte=1"`
	Y      float64 `json:"y" binding:"gte=0,lte=1"`
	Width  float64 `json:"width" binding:"required,gte=0.24,lte=0.8"`
}

type SpecificationTableStyleRequest struct {
	Style           string `json:"style" binding:"required,oneof=simple-table"`
	BorderWidth     int    `json:"border_width" binding:"required,gte=1,lte=8"`
	BorderColor     string `json:"border_color" binding:"required,hexcolor"`
	BackgroundColor string `json:"background_color" binding:"required"`
	TextColor       string `json:"text_color" binding:"required,hexcolor"`
}

// TaskSpecificationRequest 是随标准化任务一次绘制的规格表格配置。
type TaskSpecificationRequest struct {
	Specifications []SpecificationRequest            `json:"specifications"`
	TablePosition  SpecificationTablePositionRequest `json:"table_position"`
	TableStyle     SpecificationTableStyleRequest    `json:"table_style"`
}

type SizeChartRequest struct {
	AssetID        string                            `json:"asset_id" binding:"required"`
	TemplateID     string                            `json:"template_id" binding:"required"`
	Specifications []SpecificationRequest            `json:"specifications" binding:"required,min=1,max=10,dive"`
	TablePosition  SpecificationTablePositionRequest `json:"table_position" binding:"required"`
	TableStyle     SpecificationTableStyleRequest    `json:"table_style" binding:"required"`
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
