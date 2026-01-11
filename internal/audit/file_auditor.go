package audit

import (
	"encoding/json"
	"os"
	"sync"

	"go.uber.org/zap"
)

// FileAuditor реализует Observer для записи событий аудита в файл.
// Потокобезопасен благодаря использованию mutex.
type FileAuditor struct {
	filePath string
	logger   *zap.Logger
	mu       sync.Mutex
}

// NewFileAuditor создаёт новый FileAuditor для записи в указанный файл.
func NewFileAuditor(filePath string, logger *zap.Logger) *FileAuditor {
	return &FileAuditor{
		filePath: filePath,
		logger:   logger,
	}
}

// Update записывает событие аудита в файл в формате JSON (по одной строке на событие).
func (fa *FileAuditor) Update(event AuditEvent) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	file, err := os.OpenFile(fa.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fa.logger.Error("failed to open audit file",
			zap.String("path", fa.filePath),
			zap.Error(err))
		return err
	}
	defer file.Close()

	eventJSON, err := json.Marshal(event)
	if err != nil {
		fa.logger.Error("failed to marshal audit event", zap.Error(err))
		return err
	}

	if _, err := file.Write(append(eventJSON, '\n')); err != nil {
		fa.logger.Error("failed to write to audit file",
			zap.String("path", fa.filePath),
			zap.Error(err))
		return err
	}

	fa.logger.Debug("audit event written to file",
		zap.String("path", fa.filePath),
		zap.Int("metrics_count", len(event.Metrics)),
		zap.String("ip", event.IPAddress))

	return nil
}

// GetID возвращает уникальный идентификатор аудитора.
func (fa *FileAuditor) GetID() string {
	return "file_auditor_" + fa.filePath
}
