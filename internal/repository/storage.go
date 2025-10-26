package repository

import (
	"errors"
	"sync"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

var (
	ErrCounterNotFound = errors.New("counter not found")
	ErrGaugeNotFound   = errors.New("gauge not found")
)

type Storage interface {
	UpdateGauge(id string, value float64)
	UpdateCounter(id string, delta int64)
	GetGauge(id string) (float64, error)
	GetCounter(id string) (int64, error)
	GetAllMetrics() (map[string]model.Metrics, error)
}

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (ms *MemStorage) UpdateGauge(id string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauges[id] = value
}

func (ms *MemStorage) UpdateCounter(id string, delta int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counters[id] += delta
}

func (ms *MemStorage) GetGauge(id string) (float64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, ok := ms.gauges[id]
	if !ok {
		return 0, ErrGaugeNotFound
	}
	return value, nil
}

func (ms *MemStorage) GetCounter(id string) (int64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, ok := ms.counters[id]
	if !ok {
		return 0, ErrCounterNotFound
	}
	return value, nil
}

func (ms *MemStorage) GetAllMetrics() (map[string]model.Metrics, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	metrics := make(map[string]model.Metrics)
	for id, value := range ms.gauges {
		v := value
		metrics[id] = model.Metrics{
			ID:    id,
			MType: model.Gauge,
			Value: &v,
		}
	}
	for id, value := range ms.counters {
		v := value
		metrics[id] = model.Metrics{
			ID:    id,
			MType: model.Counter,
			Delta: &v,
		}
	}
	return metrics, nil
}
