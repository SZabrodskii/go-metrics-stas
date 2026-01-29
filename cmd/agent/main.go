package main

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/agent"
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/SZabrodskii/go-metrics-stas/pkg/buildinfo"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"go.uber.org/fx"
)

func main() {
	buildinfo.Print()

	fx.New(
		logging.Module,
		config.ProvideAgentConfig(),
		agent.Module,
		fx.Invoke(handler.RegisterSignalHandler),
	).Run()
}
