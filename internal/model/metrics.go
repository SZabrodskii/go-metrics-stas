package model

const (
	Counter = "counter"
	Gauge   = "gauge"
)

const (
	MetricAlloc         = "Alloc"
	MetricTotalAlloc    = "TotalAlloc"
	MetricSys           = "Sys"
	MetricLookups       = "Lookups"
	MetricMallocs       = "Mallocs"
	MetricFrees         = "Frees"
	
	MetricHeapAlloc     = "HeapAlloc"
	MetricHeapSys       = "HeapSys"
	MetricHeapIdle      = "HeapIdle"
	MetricHeapInuse     = "HeapInuse"
	MetricHeapReleased  = "HeapReleased"
	MetricHeapObjects   = "HeapObjects"
	
	MetricStackInuse    = "StackInuse"
	MetricStackSys      = "StackSys"
	
	MetricMSpanInuse    = "MSpanInuse"
	MetricMSpanSys      = "MSpanSys"
	
	MetricMCacheInuse   = "MCacheInuse"
	MetricMCacheSys     = "MCacheSys"
	
	MetricBuckHashSys   = "BuckHashSys"
	MetricGCSys         = "GCSys"
	MetricOtherSys      = "OtherSys"

	MetricNextGC        = "NextGC"
	MetricLastGC        = "LastGC"
	MetricPauseTotalNs  = "PauseTotalNs"
	MetricNumGC         = "NumGC"
	MetricNumForcedGC   = "NumForcedGC"
	MetricGCCPUFraction = "GCCPUFraction"
	
	MetricPollCount     = "PollCount"    
	MetricRandomValue   = "RandomValue"  
)

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

var CounterMetrics = []string{
	MetricPollCount,
}

var AllMetrics = append(GaugeMetrics, CounterMetrics...)

func GetMetricType(name string) string {
	for _, metric := range CounterMetrics {
		if metric == name {
			return Counter
		}
	}
	return Gauge
}

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
