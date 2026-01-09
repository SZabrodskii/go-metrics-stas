package logging

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := cfg.Build()
	if err != nil {
		t.Fatalf("Expected no error building logger, got %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	logger.Info("test message")
	logger.Sync()
}
