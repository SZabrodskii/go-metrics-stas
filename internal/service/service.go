package service

import "github.com/SZabrodskii/go-metrics-stas/internal/model"

// MetricsService определяет интерфейс сервисного слоя для работы с метриками.
// Не зависит от HTTP — может использоваться из любого транспорта (gRPC, CLI).
type MetricsService interface {
	UpdateGauge(id string, value float64) error
	UpdateCounter(id string, delta int64) (int64, error)
	GetGauge(id string) (float64, error)
	GetCounter(id string) (int64, error)
	GetAllMetrics() (map[string]model.Metrics, error)
	UpdateBatch(metrics []model.Metrics) error
}
