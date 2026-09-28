package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage 管理原图和处理结果的本地文件。
type Storage struct {
	root string
}

// New 创建本地文件存储。
func New(root string) (*Storage, error) {
	for _, dir := range []string{filepath.Join(root, "uploads"), filepath.Join(root, "outputs")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建文件目录失败: %w", err)
		}
	}
	return &Storage{root: root}, nil
}

// SaveUpload 将上传内容写入临时文件，并限制单文件大小。
func (s *Storage) SaveUpload(taskID, assetID string, source io.Reader, maxSize int64) (string, int64, error) {
	dir := filepath.Join(s.root, "uploads", taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("创建上传目录失败: %w", err)
	}
	path := filepath.Join(dir, assetID+".upload")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("创建上传文件失败: %w", err)
	}
	written, copyErr := io.Copy(file, io.LimitReader(source, maxSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("保存上传文件失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("关闭上传文件失败: %w", closeErr)
	}
	if written > maxSize {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("单张图片超过上传大小限制")
	}
	return path, written, nil
}

// FinalizeUpload 根据图片真实格式确定原图扩展名。
func (s *Storage) FinalizeUpload(tempPath, format string) (string, error) {
	ext := Extension(format)
	if ext == "" {
		return "", fmt.Errorf("不支持的图片格式")
	}
	path := tempPath[:len(tempPath)-len(filepath.Ext(tempPath))] + ext
	if err := os.Rename(tempPath, path); err != nil {
		return "", fmt.Errorf("保存原图失败: %w", err)
	}
	return path, nil
}

// OutputPath 返回任务输出文件的安全路径。
func (s *Storage) OutputPath(taskID, outputID, format string) (string, error) {
	ext := Extension(format)
	if ext == "" {
		return "", fmt.Errorf("不支持的输出格式")
	}
	dir := filepath.Join(s.root, "outputs", taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}
	return filepath.Join(dir, outputID+ext), nil
}

// RemoveTask 删除任务对应的原图和输出文件。
func (s *Storage) RemoveTask(taskID string) error {
	if err := os.RemoveAll(filepath.Join(s.root, "uploads", taskID)); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.root, "outputs", taskID))
}

// Extension 返回图片格式对应的扩展名。
func Extension(format string) string {
	switch format {
	case "jpeg", "jpg":
		return ".jpg"
	case "png":
		return ".png"
	case "webp":
		return ".webp"
	default:
		return ""
	}
}
