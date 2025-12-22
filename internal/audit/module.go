package audit

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Provide(NewPublisherWithConfig)

func NewPublisherWithConfig(cfg *config.ServerConfig, logger *zap.Logger) *Publisher {
	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		logger.Info("audit is disabled - no audit-file or audit-url configured")
		return nil
	}

	publisher := NewPublisher(logger)

	if cfg.AuditFile != "" {
		fileAuditor := NewFileAuditor(cfg.AuditFile, logger)
		publisher.Subscribe(fileAuditor)
		logger.Info("file audit enabled", zap.String("path", cfg.AuditFile))
	}

	if cfg.AuditURL != "" {
		httpAuditor := NewHTTPAuditor(cfg.AuditURL, logger)
		publisher.Subscribe(httpAuditor)
		logger.Info("HTTP audit enabled", zap.String("url", cfg.AuditURL))
	}

	return publisher
}
