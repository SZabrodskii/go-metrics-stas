package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New(lc fx.Lifecycle, cfg *config.ServerConfig, logger *zap.Logger) (*sql.DB, error) {
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
			logger.Info("Connected to database PostgreSQL",
				zap.String("db_host", "127.0.0.1"),
				zap.String("db_name", "videos"),
			)
			driver, err := postgres.WithInstance(db, &postgres.Config{})
			if err != nil {
				return err
			}
			m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
			if err != nil {
				return err
			}
			if upErr := m.Up(); upErr != nil {
				if errors.Is(upErr, migrate.ErrNoChange) {
					logger.Info("Nothing to migrate")
				} else {
					return upErr
				}
			} else {
				logger.Info("Migrated")
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})
	return db, nil
}
