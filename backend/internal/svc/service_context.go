package svc

import (
	"log/slog"

	"picflow/backend/configs"
	"picflow/backend/internal/database"
	"picflow/backend/internal/llm"
	"picflow/backend/internal/repository"
	"picflow/backend/internal/storage"
	"picflow/backend/internal/worker"

	"gorm.io/gorm"
)

// ServiceContext 保存业务处理器共享的依赖。
type ServiceContext struct {
	Config      *configs.Config
	DB          *gorm.DB
	Logger      *slog.Logger
	Tasks       *repository.TaskRepository
	Templates   *repository.TemplateRepository
	Storage     *storage.Storage
	Workers     *worker.Pool
	ImageEditor llm.ImageEditor
}

func NewServiceContext(cfg *configs.Config, logger *slog.Logger) (*ServiceContext, error) {
	editor, err := llm.NewImageEditor(cfg.LLM)
	if err != nil {
		return nil, err
	}
	db, err := database.Open(cfg.DatabasePath(), cfg.OTLPTraceEndpoint != "")
	if err != nil {
		return nil, err
	}
	files, err := storage.New(cfg.DataDir)
	if err != nil {
		_ = database.Close(db)
		return nil, err
	}
	tasks := repository.NewTaskRepository(db)
	templates := repository.NewTemplateRepository(db)
	if err = templates.RemoveLegacyBuiltIns(); err != nil {
		_ = database.Close(db)
		return nil, err
	}
	if err = tasks.FailInterrupted(); err != nil {
		_ = database.Close(db)
		return nil, err
	}
	workers := worker.New(cfg.WorkerCount, tasks, files, logger, editor)
	workers.Start()
	return &ServiceContext{
		Config: cfg, DB: db, Logger: logger, Tasks: tasks,
		Templates: templates, Storage: files, Workers: workers, ImageEditor: editor,
	}, nil
}

func (s *ServiceContext) Close() error {
	s.Workers.Close()
	return database.Close(s.DB)
}
