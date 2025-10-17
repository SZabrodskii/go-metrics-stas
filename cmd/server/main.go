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
		config.ProvideServerConfig(),
		fx.Provide(
			fx.Annotate(
				repository.NewFileStorage,
				fx.As(new(repository.Storage)),
			),
			handler.NewMetricsHandler,
			server.NewRouter,
		),
		fx.Invoke(
			server.NewServer,
		),
	).Run()
}
