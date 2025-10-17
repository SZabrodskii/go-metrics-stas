package agent

import (
	"math/rand"
	"runtime"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

type metricsCollector struct {
	PollCount int64
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{}
}

func (mc *metricsCollector) CollectMetrics() map[string]model.Metrics {
	mc.PollCount++

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := make(map[string]model.Metrics)

	for _, metricName := range model.GaugeMetrics {
		if metricName == model.MetricPollCount || metricName == model.MetricRandomValue {
			continue
		}
		var value float64
		switch metricName {
		case model.MetricAlloc:
			value = float64(memStats.Alloc)
		case model.MetricTotalAlloc:
			value = float64(memStats.TotalAlloc)
		case model.MetricSys:
			value = float64(memStats.Sys)
		case model.MetricNumGC:
			value = float64(memStats.NumGC)
		case model.MetricLookups:
			value = float64(memStats.Lookups)
		case model.MetricMallocs:
			value = float64(memStats.Mallocs)
		case model.MetricFrees:
			value = float64(memStats.Frees)
		case model.MetricHeapAlloc:
			value = float64(memStats.HeapAlloc)
		case model.MetricHeapSys:
			value = float64(memStats.HeapSys)
		case model.MetricHeapIdle:
			value = float64(memStats.HeapIdle)
		case model.MetricHeapInuse:
			value = float64(memStats.HeapInuse)
		case model.MetricHeapReleased:
			value = float64(memStats.HeapReleased)
		case model.MetricHeapObjects:
			value = float64(memStats.HeapObjects)
		case model.MetricStackInuse:
			value = float64(memStats.StackInuse)
		case model.MetricStackSys:
			value = float64(memStats.StackSys)
		case model.MetricMSpanInuse:
			value = float64(memStats.MSpanInuse)
		case model.MetricMSpanSys:
			value = float64(memStats.MSpanSys)
		case model.MetricMCacheInuse:
			value = float64(memStats.MCacheInuse)
		case model.MetricMCacheSys:
			value = float64(memStats.MCacheSys)
		case model.MetricBuckHashSys:
			value = float64(memStats.BuckHashSys)
		case model.MetricGCSys:
			value = float64(memStats.GCSys)
		case model.MetricOtherSys:
			value = float64(memStats.OtherSys)
		case model.MetricNextGC:
			value = float64(memStats.NextGC)
		case model.MetricLastGC:
			value = float64(memStats.LastGC)
		case model.MetricPauseTotalNs:
			value = float64(memStats.PauseTotalNs)
		case model.MetricNumForcedGC:
			value = float64(memStats.NumForcedGC)
		case model.MetricGCCPUFraction:
			value = memStats.GCCPUFraction
		default:
			continue
		}

		metrics[metricName] = model.Metrics{
			ID:    metricName,
			MType: model.Gauge,
			Value: &value,
		}
	}

	pollCount := mc.PollCount
	randomValue := rand.Float64() * 100

	metrics[model.MetricPollCount] = model.Metrics{
		ID:    model.MetricPollCount,
		MType: model.Counter,
		Delta: &pollCount,
	}
	metrics[model.MetricRandomValue] = model.Metrics{
		ID:    model.MetricRandomValue,
		MType: model.Gauge,
		Value: &randomValue,
	}

	return metrics
}
