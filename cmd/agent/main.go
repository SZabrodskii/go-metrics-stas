package main

import (
	"context"

	"github.com/SZabrodskii/go-metrics-stas/internal/agent"
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		logging.Module,
		fx.Provide(
			config.NewAgentConfig,
			func(cfg *config.AgentConfig) *agent.Agent {
				return agent.NewAgent(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval)
			},
		),
		fx.Invoke(runAgent),
	).Run()

}

func runAgent(lc fx.Lifecycle, agent *agent.Agent, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := agent.Run(ctx); err != nil && err != context.Canceled {
					logger.Fatal("Failed to start agent", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("Agent stopped")
			cancel()
			return nil
		},
	})
}
