package repository

import (
	"picflow/backend/internal/model"

	"gorm.io/gorm"
)

// TemplateRepository 负责处理模板的持久化。
type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

// RemoveLegacyBuiltIns 清理旧版四个预置模板，不影响用户自建模板和任务记录。
func (r *TemplateRepository) RemoveLegacyBuiltIns() error {
	return r.db.Where("built_in = ? AND id IN ?", true, []string{"square-white", "square-transparent", "portrait", "size-standard"}).Delete(&model.Template{}).Error
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
