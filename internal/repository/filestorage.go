package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fileBackedStorage struct {
	*MemStorage

	filePath  string
	interval  time.Duration
	logger    *zap.Logger
	syncWrite bool

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewFileStorage(lc fx.Lifecycle, cfg *config.ServerConfig, logger *zap.Logger) (Storage, error) {
	ms := NewMemStorage()
	s := &fileBackedStorage{
		MemStorage: ms,
		filePath:   cfg.FileStoragePath,
		interval:   cfg.StoreInterval,
		logger:     logger,
		syncWrite:  cfg.StoreInterval == 0,
		stopCh:     make(chan struct{}),
	}

	if cfg.Restore {
		if err := s.loadFromFile(); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.Error("error loading metrics from file", zap.Error(err))
			return nil, err
		}
		logger.Info("loaded metrics from file", zap.String("path", cfg.FileStoragePath))
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if s.interval > 0 {
				t := time.NewTicker(s.interval)
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					for {
						select {
						case <-t.C:
							if err := s.saveToFile(); err != nil {
								s.logger.Error("periodic save failed", zap.Error(err), zap.String("path", s.filePath))
							}
						case <-s.stopCh:
							t.Stop()
							return
						}
					}
				}()
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(s.stopCh)
			done := make(chan struct{})
			go func() { s.wg.Wait(); close(done) }()

			select {
			case <-done:
			case <-ctx.Done():
			}
			return s.saveToFile()
		},
	})
	return s, nil

}

func (s *fileBackedStorage) UpdateGauge(id string, value float64) {
	s.MemStorage.UpdateGauge(id, value)
	if s.syncWrite {
		if err := s.saveToFile(); err != nil {
			s.logger.Error("sync save failed", zap.Error(err), zap.String("path", s.filePath))
		}
	}
}

func (s *fileBackedStorage) UpdateCounter(id string, delta int64) {
	s.MemStorage.UpdateCounter(id, delta)
	if s.syncWrite {
		if err := s.saveToFile(); err != nil {
			s.logger.Error("sync save failed", zap.Error(err), zap.String("path", s.filePath))
		}
	}
}

func (s *fileBackedStorage) saveToFile() error {
	m, err := s.GetAllMetrics()
	if err != nil {
		return err
	}
	list := make([]model.Metrics, 0, len(m))
	for _, v := range m {
		list = append(list, v)
	}

	if err = os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil && !os.IsExist(err) {
		return err
	}
	tmp := s.filePath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err = enc.Encode(list); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func (s *fileBackedStorage) loadFromFile() error {
	f, err := os.Open(s.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var list []model.Metrics
	if err = json.NewDecoder(f).Decode(&list); err != nil {
		return err
	}

	for _, m := range list {
		switch m.MType {
		case "gauge":
			if m.Value != nil {
				s.MemStorage.UpdateGauge(m.ID, *m.Value)
			}
		case "counter":
			if m.Delta != nil {
				s.MemStorage.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}

func (s *fileBackedStorage) UpdateBatch(items []model.Metrics) error {
	if len(items) == 0 {
		return nil
	}

	_ = s.MemStorage.UpdateBatch(items)

	if s.syncWrite {
		if err := s.saveToFile(); err != nil {
			s.logger.Error("sync save failed (batch)", zap.Error(err), zap.String("path", s.filePath))
			return err
		}
	}
	return nil
}
