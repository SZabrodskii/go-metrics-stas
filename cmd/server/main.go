package main

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/audit"
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/config/db"
	"github.com/SZabrodskii/go-metrics-stas/internal/grpcserver"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/SZabrodskii/go-metrics-stas/internal/server"
	"github.com/SZabrodskii/go-metrics-stas/internal/service"
	"github.com/SZabrodskii/go-metrics-stas/pkg/buildinfo"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"go.uber.org/fx"
)

func main() {
	buildinfo.Print()

	fx.New(
		logging.Module,
		audit.Module,
		config.ProvideServerConfig(),
		fx.Provide(
			db.New,
			repository.NewStorage,
			service.NewMetricsService,
			handler.NewMetricsHandler,
			handler.NewPingHandler,
			server.NewRouter,
			grpcserver.NewGRPCServer,
		),
		fx.Invoke(
			server.NewServer,
			grpcserver.RunGRPCServer,
			handler.RegisterSignalHandler,
		),
	).Run()
}
