package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"picflow/backend/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// Open 创建本地 SQLite 连接并初始化 PicFlow 所需数据表。
func Open(path string, tracingEnabled bool) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库失败: %w", err)
	}
	if tracingEnabled {
		if err = db.Use(tracing.NewPlugin(tracing.WithoutQueryVariables())); err != nil {
			return nil, fmt.Errorf("启用数据库链路追踪失败: %w", err)
		}
	}
	if err = normalizePathColumns(db); err != nil {
		return nil, err
	}
	if err = db.AutoMigrate(&model.Task{}, &model.Asset{}, &model.Output{}, &model.Template{}, &model.SizeAnnotation{}); err != nil {
		return nil, fmt.Errorf("初始化数据库表失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}
	// SQLite 写入需要串行执行，避免同一进程内出现锁竞争。
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("检查数据库连接失败: %w", err)
	}
	return db, nil
}

// Close 关闭底层数据库连接。
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// normalizePathColumns 兼容早期 PicFlow 使用过的 source_path 和 output_path 列名。
func normalizePathColumns(db *gorm.DB) error {
	for _, item := range []struct {
		model any
		old   string
	}{
		{model: &model.Asset{}, old: "source_path"},
		{model: &model.Output{}, old: "output_path"},
	} {
		migrator := db.Migrator()
		if !migrator.HasTable(item.model) || !migrator.HasColumn(item.model, item.old) {
			continue
		}
		if !migrator.HasColumn(item.model, "path") {
			if err := migrator.RenameColumn(item.model, item.old, "path"); err != nil {
				return fmt.Errorf("迁移文件路径字段失败: %w", err)
			}
			continue
		}
		if err := migrator.DropColumn(item.model, item.old); err != nil {
			return fmt.Errorf("清理旧文件路径字段失败: %w", err)
		}
	}
	return nil
}
