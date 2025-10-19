package main

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/agent"
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		logging.Module,
		config.ProvideAgentConfig(),
		agent.Module,
	).Run()

}
