package model

import "time"

const (
	TaskStatusQueued     = "queued"
	TaskStatusProcessing = "processing"
	TaskStatusSucceeded  = "succeeded"
	TaskStatusFailed     = "failed"

	OutputTypeStandardized = "standardized"
	OutputTypeSizeChart    = "size_chart"
)

// Task 表示一次批量图片处理任务。
type Task struct {
	ID              string   `gorm:"primaryKey;size:36"`
	Type            string   `gorm:"size:32;not null"`
	Status          string   `gorm:"size:20;not null;index"`
	ErrorMessage    string   `gorm:"type:text"`
	ConfigJSON      string   `gorm:"type:text;not null"`
	TotalAssets     int      `gorm:"not null;default:0"`
	CompletedAssets int      `gorm:"not null;default:0"`
	Assets          []Asset  `gorm:"constraint:OnDelete:CASCADE"`
	Outputs         []Output `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Asset 保存任务中的原始图片信息，原图文件不会被覆盖。
type Asset struct {
	ID         string `gorm:"primaryKey;size:36"`
	TaskID     string `gorm:"size:36;not null;index"`
	Filename   string `gorm:"type:text;not null"`
	SourcePath string `gorm:"column:path;type:text;not null"`
	Width      int    `gorm:"not null"`
	Height     int    `gorm:"not null"`
	Format     string `gorm:"size:16;not null"`
	Size       int64  `gorm:"not null"`
	CreatedAt  time.Time
}

// Output 保存标准化图片或尺寸图的输出信息。
type Output struct {
	ID         string `gorm:"primaryKey;size:36"`
	TaskID     string `gorm:"size:36;not null;index"`
	AssetID    string `gorm:"size:36;not null;index"`
	Type       string `gorm:"size:24;not null"`
	Filename   string `gorm:"type:text;not null"`
	OutputPath string `gorm:"column:path;type:text;not null"`
	Width      int    `gorm:"not null"`
	Height     int    `gorm:"not null"`
	Format     string `gorm:"size:16;not null"`
	Size       int64  `gorm:"not null"`
	CreatedAt  time.Time
}

// Template 保存可复用的图片处理参数。
type Template struct {
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"size:80;not null"`
	Type        string `gorm:"size:24;not null;index"`
	Description string `gorm:"size:255"`
	ConfigJSON  string `gorm:"type:text;not null"`
	BuiltIn     bool   `gorm:"not null;default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SizeAnnotation 保存用户为尺寸图输入的真实尺寸。
type SizeAnnotation struct {
	ID         string  `gorm:"primaryKey;size:36"`
	TaskID     string  `gorm:"size:36;not null;index"`
	AssetID    string  `gorm:"size:36;not null;index"`
	OutputID   string  `gorm:"size:36;not null;index"`
	Width      float64 `gorm:"not null"`
	Height     float64 `gorm:"not null"`
	Depth      float64 `gorm:"not null;default:0"`
	Unit       string  `gorm:"size:8;not null"`
	TemplateID string  `gorm:"size:64;not null"`
	CreatedAt  time.Time
}
