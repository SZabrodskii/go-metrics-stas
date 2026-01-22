// Package main реализует статический анализатор кода для проверки
// использования panic, log.Fatal, os.Exit и zap.Logger.Fatal.
package main

import (
	"github.com/SZabrodskii/go-metrics-stas/cmd/linter/exitcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(exitcheck.Analyzer)
}
