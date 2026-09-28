package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const logDateLayout = "2006-01-02"

// DailyWriter 按自然日将日志写入不同文件，并清理超过保留期限的日志。
type DailyWriter struct {
	mu            sync.Mutex
	directory     string
	retentionDays int
	currentDate   string
	file          *os.File
}

// NewDailyWriter 创建按天轮转的日志写入器。
func NewDailyWriter(directory string, retentionDays int) (*DailyWriter, error) {
	if retentionDays < 1 {
		return nil, fmt.Errorf("log retention days must be greater than zero")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	writer := &DailyWriter{
		directory:     directory,
		retentionDays: retentionDays,
	}
	if err := writer.rotate(time.Now()); err != nil {
		return nil, err
	}
	return writer, nil
}

// Write 写入当天的日志文件，跨天后的第一条日志会触发文件轮转。
func (w *DailyWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	if now.Format(logDateLayout) != w.currentDate {
		if err := w.rotate(now); err != nil {
			return 0, err
		}
	}
	return w.file.Write(data)
}

// Close 关闭当前日志文件。
func (w *DailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *DailyWriter) rotate(now time.Time) error {
	date := now.Format(logDateLayout)
	path := filepath.Join(w.directory, date+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = file
	w.currentDate = date

	if err := w.removeExpired(now); err != nil {
		return fmt.Errorf("remove expired log files: %w", err)
	}
	return nil
}

func (w *DailyWriter) removeExpired(now time.Time) error {
	entries, err := os.ReadDir(w.directory)
	if err != nil {
		return err
	}

	location := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	oldestRetainedDate := today.AddDate(0, 0, -(w.retentionDays - 1))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue
		}
		date, err := time.ParseInLocation(logDateLayout, strings.TrimSuffix(entry.Name(), ".log"), location)
		if err != nil || !date.Before(oldestRetainedDate) {
			continue
		}
		if err = os.Remove(filepath.Join(w.directory, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
