package service

import (
	"errors"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"go.uber.org/zap"
)

type metricsService struct {
	repo   repository.Storage
	logger *zap.Logger
}

// NewMetricsService создаёт новый экземпляр MetricsService.
func NewMetricsService(repo repository.Storage, logger *zap.Logger) MetricsService {
	return &metricsService{
		repo:   repo,
		logger: logger,
	}
}

func (s *metricsService) UpdateGauge(id string, value float64) error {
	if id == "" {
		return ErrInvalidMetricID
	}
	s.repo.UpdateGauge(id, value)
	return nil
}

func (s *metricsService) UpdateCounter(id string, delta int64) (int64, error) {
	if id == "" {
		return 0, ErrInvalidMetricID
	}
	s.repo.UpdateCounter(id, delta)
	newVal, err := s.repo.GetCounter(id)
	if err != nil {
		return 0, err
	}
	return newVal, nil
}

func (s *metricsService) GetGauge(id string) (float64, error) {
	if id == "" {
		return 0, ErrInvalidMetricID
	}
	val, err := s.repo.GetGauge(id)
	if err != nil {
		if errors.Is(err, repository.ErrGaugeNotFound) {
			return 0, ErrMetricNotFound
		}
		return 0, err
	}
	return val, nil
}

func (s *metricsService) GetCounter(id string) (int64, error) {
	if id == "" {
		return 0, ErrInvalidMetricID
	}
	val, err := s.repo.GetCounter(id)
	if err != nil {
		if errors.Is(err, repository.ErrCounterNotFound) {
			return 0, ErrMetricNotFound
		}
		return 0, err
	}
	return val, nil
}

func (s *metricsService) GetAllMetrics() (map[string]model.Metrics, error) {
	return s.repo.GetAllMetrics()
}

func (s *metricsService) UpdateBatch(metrics []model.Metrics) error {
	for _, m := range metrics {
		if m.ID == "" {
			return ErrInvalidMetricID
		}
		switch m.MType {
		case "gauge":
			if m.Value == nil {
				return ErrMissingValue
			}
			s.repo.UpdateGauge(m.ID, *m.Value)
		case "counter":
			if m.Delta == nil {
				return ErrMissingDelta
			}
			s.repo.UpdateCounter(m.ID, *m.Delta)
		default:
			return ErrInvalidMetricType
		}
	}
	return nil
}
