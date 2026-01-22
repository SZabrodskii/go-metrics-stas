package exitcheck_test

import (
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/cmd/linter/exitcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitCheck(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, exitcheck.Analyzer, "panic", "logfatal", "osexit", "mainpkg")
}
