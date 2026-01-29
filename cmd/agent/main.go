package main

import (
	"fmt"

	"github.com/SZabrodskii/go-metrics-stas/internal/agent"
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/SZabrodskii/go-metrics-stas/pkg/buildinfo"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"go.uber.org/fx"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

func main() {
	buildinfo.Print()

	fx.New(
		logging.Module,
		config.ProvideAgentConfig(),
		agent.Module,
		fx.Invoke(handler.RegisterSignalHandler),
	).Run()
}
