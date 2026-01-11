// Package logging предоставляет конфигурацию логгера для приложения.
package logging

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module предоставляет fx модуль для внедрения zap.Logger.
var Module = fx.Provide(NewLogger)

// NewLogger создаёт production zap.Logger с автоматическим Sync при остановке.
func NewLogger(lc fx.Lifecycle) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			_ = l.Sync()
			return nil
		},
	})
	return l, nil
}
