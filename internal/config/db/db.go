package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewDB(lc fx.Lifecycle, cfg *config.ServerConfig, logger *zap.Logger) (*sql.DB, error) {
	if cfg.DatabaseDSN == "" {
		logger.Warn("Database DSN not set, using in-memory store")
		return nil, nil
	}

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 30)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err != nil {
				return err
			}
			logger.Info("Connected to database",
				zap.String("db_host", "127.0.0.1"),
				zap.String("db_name", "videos"),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})
	return db, nil
}
