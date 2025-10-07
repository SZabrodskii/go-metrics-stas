package logging

import "go.uber.org/zap"

func NewLogger() (*zap.Logger, func()) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	l, _ := cfg.Build()
	cleanup := func() {
		_ = l.Sync()
	}
	return l, cleanup
}
