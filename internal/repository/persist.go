package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewPersister(lc fx.Lifecycle, storage Storage, cfg *config.ServerConfig, logger *zap.Logger) {
	if cfg.Restore {
		if err := loadFromFile(storage, cfg.FileStoragePath); err != nil {
			if !os.IsNotExist(err) {
				logger.Error("failed to load from file", zap.String("path", cfg.FileStoragePath), zap.Error(err))
			}
		} else {
			logger.Info("successfully loaded from file", zap.String("path", cfg.FileStoragePath))
		}
	}

	var ticker *time.Ticker

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if cfg.StoreInterval > 0 {
				ticker = time.NewTicker(cfg.StoreInterval)
				go func() {
					for range ticker.C {
						if err := saveToFile(storage, cfg.FileStoragePath); err != nil {
							logger.Error("failed to save to file", zap.String("path", cfg.FileStoragePath), zap.Error(err))
						}
					}
				}()
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if ticker != nil {
				ticker.Stop()
			}
			if err := saveToFile(storage, cfg.FileStoragePath); err != nil {
				logger.Error("failed to save to file", zap.String("path", cfg.FileStoragePath), zap.Error(err))
				return err
			}
			logger.Info("successfully saved to file", zap.String("path", cfg.FileStoragePath))
			return nil
		},
	})
}

func SyncSave(storage Storage, cfg *config.ServerConfig, logger *zap.Logger) {
	if cfg.StoreInterval == 0 {
		if err := saveToFile(storage, cfg.FileStoragePath); err != nil {
			logger.Error("error sync saving metrics", zap.String("path", cfg.FileStoragePath), zap.Error(err))
		}
	}
}

func exportAll(s Storage) ([]model.Metrics, error) {
	m, err := s.GetAllMetrics()
	if err != nil {
		return nil, err
	}
	out := make([]model.Metrics, 0, len(m))

	for _, v := range m {
		out = append(out, v)
	}
	return out, nil
}

func saveToFile(s Storage, path string) error {
	list, err := exportAll(s)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err = enc.Encode(list); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadFromFile(s Storage, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var list []model.Metrics
	dec := json.NewDecoder(f)
	if err = dec.Decode(&list); err != nil {
		return err
	}

	for _, m := range list {
		switch m.MType {
		case model.Gauge:
			if m.Value != nil {
				s.UpdateGauge(m.ID, *m.Value)
			}
		case model.Counter:
			if m.Delta != nil {
				s.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}
