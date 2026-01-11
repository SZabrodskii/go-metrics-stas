// Package model содержит модели данных для системы сбора метрик.
package model

// Типы метрик, поддерживаемые системой.
const (
	// Counter представляет метрику-счётчик, значение которой накапливается.
	Counter = "counter"
	// Gauge представляет метрику-измеритель с произвольным значением.
	Gauge = "gauge"
)

// Константы имён метрик, собираемых из runtime.MemStats.
const (
	MetricAlloc      = "Alloc"
	MetricTotalAlloc = "TotalAlloc"
	MetricSys        = "Sys"
	MetricLookups    = "Lookups"
	MetricMallocs    = "Mallocs"
	MetricFrees      = "Frees"

	MetricHeapAlloc    = "HeapAlloc"
	MetricHeapSys      = "HeapSys"
	MetricHeapIdle     = "HeapIdle"
	MetricHeapInuse    = "HeapInuse"
	MetricHeapReleased = "HeapReleased"
	MetricHeapObjects  = "HeapObjects"

	MetricStackInuse = "StackInuse"
	MetricStackSys   = "StackSys"

	MetricMSpanInuse = "MSpanInuse"
	MetricMSpanSys   = "MSpanSys"

	MetricMCacheInuse = "MCacheInuse"
	MetricMCacheSys   = "MCacheSys"

	MetricBuckHashSys = "BuckHashSys"
	MetricGCSys       = "GCSys"
	MetricOtherSys    = "OtherSys"

	MetricNextGC        = "NextGC"
	MetricLastGC        = "LastGC"
	MetricPauseTotalNs  = "PauseTotalNs"
	MetricNumGC         = "NumGC"
	MetricNumForcedGC   = "NumForcedGC"
	MetricGCCPUFraction = "GCCPUFraction"

	MetricPollCount   = "PollCount"
	MetricRandomValue = "RandomValue"
)

// GaugeMetrics содержит список всех gauge метрик, собираемых агентом.
var GaugeMetrics = []string{
	MetricAlloc, MetricTotalAlloc, MetricSys, MetricLookups, MetricMallocs, MetricFrees,
	MetricHeapAlloc, MetricHeapSys, MetricHeapIdle, MetricHeapInuse, MetricHeapReleased, MetricHeapObjects,
	MetricStackInuse, MetricStackSys,
	MetricMSpanInuse, MetricMSpanSys,
	MetricMCacheInuse, MetricMCacheSys,
	MetricBuckHashSys, MetricGCSys, MetricOtherSys,
	MetricNextGC, MetricLastGC, MetricPauseTotalNs, MetricNumGC, MetricNumForcedGC, MetricGCCPUFraction,
	MetricRandomValue,
}

// CounterMetrics содержит список всех counter метрик, собираемых агентом.
var CounterMetrics = []string{
	MetricPollCount,
}

// AllMetrics содержит объединённый список всех метрик (gauge и counter).
var AllMetrics = append(GaugeMetrics, CounterMetrics...)

// GetMetricType возвращает тип метрики по её имени.
// Возвращает Counter для счётчиков и Gauge для всех остальных метрик.
func GetMetricType(name string) string {
	for _, metric := range CounterMetrics {
		if metric == name {
			return Counter
		}
	}
	return Gauge
}

// Metrics представляет единицу метрики для передачи между агентом и сервером.
// Использует плоскую структуру без вложенности для простоты сериализации.
// Delta и Value объявлены через указатели для различения нулевого значения
// и отсутствующего поля при JSON-сериализации.
type Metrics struct {
	// ID — уникальный идентификатор (имя) метрики.
	ID string `json:"id"`
	// MType — тип метрики: "counter" или "gauge".
	MType string `json:"type"`
	// Delta — значение счётчика (только для counter).
	Delta *int64 `json:"delta,omitempty"`
	// Value — значение измерения (только для gauge).
	Value *float64 `json:"value,omitempty"`
	// Hash — HMAC-подпись для верификации данных.
	Hash string `json:"hash,omitempty"`
}
