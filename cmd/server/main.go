package main

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/SZabrodskii/go-metrics-stas/internal/server"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		logging.Module,
		fx.Provide(
			config.NewServerConfig,
			repository.NewMemStorage,
			handler.NewMetricsHandler,
			server.NewRouter,
		),
		fx.Invoke(
			server.NewServer,
			repository.NewPersister,
		),
	).Run()
}
