package repository

import (
	"database/sql"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewStorage(lc fx.Lifecycle, db *sql.DB, cfg *config.ServerConfig, logger *zap.Logger) (Storage, error) {
	if db != nil {
		logger.Info("Using database connection. Storage: PostgreSQL")
		return newPostgresStorage(db), nil
	}

	if cfg.FileStoragePath != "" {
		logger.Info("Using filesystem storage (file storage). Storage: Filesystem", zap.String("path", cfg.FileStoragePath))
		return NewFileStorage(lc, cfg, logger)
	}

	logger.Info("Using memory storage (Memory). Storage: Memory")
	return NewMemStorage(), nil
}
