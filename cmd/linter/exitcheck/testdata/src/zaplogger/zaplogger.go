// Package zaplogger содержит тестовые кейсы для проверки детекции zap.Logger.Fatal.
package zaplogger

import "go.uber.org/zap"

func testZapFatal(logger *zap.Logger) {
	logger.Fatal("fatal error")
}

func testZapInfo(logger *zap.Logger) {
	logger.Info("info message")
}

func testZapError(logger *zap.Logger) {
	logger.Error("error message")
}

func testSugaredFatal(sugar *zap.SugaredLogger) {
	sugar.Fatal("fatal error")
}
