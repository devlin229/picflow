package repository

import (
	"picflow/backend/internal/model"

	"gorm.io/gorm"
)

// TaskRepository 负责处理任务、原图和输出记录的持久化。
type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *model.Task) error {
	return r.db.Create(task).Error
}

func (r *TaskRepository) Get(id string) (*model.Task, error) {
	var task model.Task
	err := r.db.Preload("Assets").Preload("Outputs").First(&task, "id = ?", id).Error
	return &task, err
}

func (r *TaskRepository) UpdateStatus(id, status, message string) error {
	return r.db.Model(&model.Task{}).Where("id = ?", id).Updates(map[string]any{
		"status": status, "error_message": message,
	}).Error
}

func (r *TaskRepository) UpdateProgress(id string, completed int) error {
	return r.db.Model(&model.Task{}).Where("id = ?", id).Update("completed_assets", completed).Error
}

func (r *TaskRepository) AddOutput(output *model.Output) error {
	return r.db.Create(output).Error
}

// DeleteStandardizedOutputs 删除任务已生成的标准化结果，并返回需要清理的文件。
func (r *TaskRepository) DeleteStandardizedOutputs(taskID string) ([]model.Output, error) {
	var outputs []model.Output
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ? AND type = ?", taskID, model.OutputTypeStandardized).Find(&outputs).Error; err != nil {
			return err
		}
		return tx.Where("task_id = ? AND type = ?", taskID, model.OutputTypeStandardized).Delete(&model.Output{}).Error
	})
	return outputs, err
}

// AddSizeResult 在一个事务中保存尺寸图输出和标注参数。
func (r *TaskRepository) AddSizeResult(output *model.Output, annotation *model.SizeAnnotation) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(output).Error; err != nil {
			return err
		}
		return tx.Create(annotation).Error
	})
}

func (r *TaskRepository) GetAsset(taskID, assetID string) (*model.Asset, error) {
	var asset model.Asset
	err := r.db.First(&asset, "id = ? AND task_id = ?", assetID, taskID).Error
	return &asset, err
}

func (r *TaskRepository) GetOutput(outputID string) (*model.Output, error) {
	var output model.Output
	err := r.db.First(&output, "id = ?", outputID).Error
	return &output, err
}

func (r *TaskRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", id).Delete(&model.SizeAnnotation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", id).Delete(&model.Output{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ?", id).Delete(&model.Asset{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Task{}, "id = ?", id).Error
	})
}

// DeleteAsset 删除原图以及由它产生的输出记录，并返回需要删除的文件路径。
func (r *TaskRepository) DeleteAsset(taskID, assetID string) (*model.Asset, []model.Output, error) {
	asset, err := r.GetAsset(taskID, assetID)
	if err != nil {
		return nil, nil, err
	}
	var outputs []model.Output
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ? AND asset_id = ?", taskID, assetID).Find(&outputs).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ? AND asset_id = ?", taskID, assetID).Delete(&model.SizeAnnotation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("task_id = ? AND asset_id = ?", taskID, assetID).Delete(&model.Output{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Asset{}, "id = ? AND task_id = ?", assetID, taskID).Error
	})
	return asset, outputs, err
}

// FailInterrupted 将服务重启前未完成的任务标记为失败，避免永久停留在处理中。
func (r *TaskRepository) FailInterrupted() error {
	return r.db.Model(&model.Task{}).
		Where("status IN ?", []string{model.TaskStatusQueued, model.TaskStatusProcessing}).
		Updates(map[string]any{"status": model.TaskStatusFailed, "error_message": "服务重启，任务未完成，请重新提交"}).Error
}
