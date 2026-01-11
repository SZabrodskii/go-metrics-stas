// Package repository предоставляет интерфейсы и реализации хранилища метрик.
package repository

import (
	"errors"
	"sync"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

// Ошибки, возвращаемые при отсутствии метрики в хранилище.
var (
	// ErrCounterNotFound возвращается при запросе несуществующего counter.
	ErrCounterNotFound = errors.New("counter not found")
	// ErrGaugeNotFound возвращается при запросе несуществующего gauge.
	ErrGaugeNotFound = errors.New("gauge not found")
)

// Storage определяет интерфейс хранилища метрик.
// Реализации должны быть потокобезопасными.
type Storage interface {
	// UpdateGauge устанавливает значение gauge метрики.
	UpdateGauge(id string, value float64)
	// UpdateCounter увеличивает значение counter метрики на delta.
	UpdateCounter(id string, delta int64)
	// GetGauge возвращает значение gauge метрики или ErrGaugeNotFound.
	GetGauge(id string) (float64, error)
	// GetCounter возвращает значение counter метрики или ErrCounterNotFound.
	GetCounter(id string) (int64, error)
	// GetAllMetrics возвращает все метрики в виде map[id]Metrics.
	GetAllMetrics() (map[string]model.Metrics, error)
	// UpdateBatch обновляет несколько метрик за одну операцию.
	UpdateBatch(items []model.Metrics) error
}

// MemStorage реализует Storage с хранением метрик в памяти.
// Потокобезопасен благодаря использованию sync.RWMutex.
type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	mu       sync.RWMutex
}

// NewMemStorage создаёт новый экземпляр MemStorage.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge устанавливает значение gauge метрики.
func (ms *MemStorage) UpdateGauge(id string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauges[id] = value
}

// UpdateCounter увеличивает значение counter метрики на delta.
func (ms *MemStorage) UpdateCounter(id string, delta int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counters[id] += delta
}

// GetGauge возвращает значение gauge метрики.
// Возвращает ErrGaugeNotFound, если метрика не существует.
func (ms *MemStorage) GetGauge(id string) (float64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, ok := ms.gauges[id]
	if !ok {
		return 0, ErrGaugeNotFound
	}
	return value, nil
}

// GetCounter возвращает значение counter метрики.
// Возвращает ErrCounterNotFound, если метрика не существует.
func (ms *MemStorage) GetCounter(id string) (int64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, ok := ms.counters[id]
	if !ok {
		return 0, ErrCounterNotFound
	}
	return value, nil
}

// GetAllMetrics возвращает все метрики в виде map[id]Metrics.
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

// UpdateBatch обновляет несколько метрик за одну атомарную операцию.
func (ms *MemStorage) UpdateBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, m := range metrics {
		switch m.MType {
		case model.Gauge:
			if m.Value != nil {
				ms.gauges[m.ID] = *m.Value
			}
		case model.Counter:
			if m.Delta != nil {
				ms.counters[m.ID] += *m.Delta
			}
		}
	}
	return nil
}
