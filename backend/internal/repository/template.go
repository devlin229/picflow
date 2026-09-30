package repository

import (
	"picflow/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TemplateRepository 负责处理模板的持久化。
type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) SeedBuiltIns() error {
	values := []model.Template{
		{ID: "square-white", Name: "白底正方形主图", Type: "main_image", Description: "1000 × 1000 · 白底 · JPG", ConfigJSON: `{"canvas_width":1000,"canvas_height":1000,"background":"#FFFFFF","layout_mode":"contain","keep_subject_complete":true,"output_format":"jpeg","margin_mode":"auto","margin":80,"output_quality":90,"replace_simple_background":false,"background_tolerance":12}`, BuiltIn: true},
		{ID: "square-transparent", Name: "透明底商品图", Type: "main_image", Description: "1000 × 1000 · 单一背景移除 · PNG", ConfigJSON: `{"canvas_width":1000,"canvas_height":1000,"background":"transparent","layout_mode":"contain","keep_subject_complete":true,"output_format":"png","margin_mode":"auto","margin":80,"output_quality":90,"replace_simple_background":true,"background_tolerance":12}`, BuiltIn: true},
		{ID: "portrait", Name: "竖版商品图", Type: "main_image", Description: "1000 × 1250 · 白底 · JPG", ConfigJSON: `{"canvas_width":1000,"canvas_height":1250,"background":"#FFFFFF","layout_mode":"contain","keep_subject_complete":true,"output_format":"jpeg","margin_mode":"auto","margin":80,"output_quality":90,"replace_simple_background":false,"background_tolerance":12}`, BuiltIn: true},
		{ID: "size-standard", Name: "标准规格图", Type: "size_chart", Description: "1000 × 1000 · 商品规格表格", ConfigJSON: `{"canvas_width":1000,"canvas_height":1000,"background":"#FFFFFF","output_format":"jpeg","margin":80,"annotation_color":"#2563EB","annotation_style":"specification_table"}`, BuiltIn: true},
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "type", "description", "config_json", "built_in", "updated_at"}),
	}).Create(&values).Error
}

func (r *TemplateRepository) List() ([]model.Template, error) {
	var values []model.Template
	err := r.db.Order("built_in DESC, created_at ASC").Find(&values).Error
	return values, err
}

func (r *TemplateRepository) Get(id string) (*model.Template, error) {
	var value model.Template
	err := r.db.First(&value, "id = ?", id).Error
	return &value, err
}

func (r *TemplateRepository) Create(value *model.Template) error {
	return r.db.Create(value).Error
}

func (r *TemplateRepository) Delete(id string) error {
	result := r.db.Where("id = ? AND built_in = ?", id, false).Delete(&model.Template{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
