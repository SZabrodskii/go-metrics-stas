package logging

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Provide(NewLogger)

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
