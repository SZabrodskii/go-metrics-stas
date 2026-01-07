package repository

import (
	"fmt"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

func BenchmarkUpdateGauge(b *testing.B) {
	storage := NewMemStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.UpdateGauge("testGauge", float64(i))
	}
}

func BenchmarkUpdateCounter(b *testing.B) {
	storage := NewMemStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.UpdateCounter("testCounter", 1)
	}
}

func BenchmarkGetGauge(b *testing.B) {
	storage := NewMemStorage()
	storage.UpdateGauge("testGauge", 123.45)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetGauge("testGauge")
	}
}

func BenchmarkGetCounter(b *testing.B) {
	storage := NewMemStorage()
	storage.UpdateCounter("testCounter", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetCounter("testCounter")
	}
}

func BenchmarkGetAllMetrics(b *testing.B) {
	storage := NewMemStorage()

	for i := 0; i < 25; i++ {
		storage.UpdateGauge(fmt.Sprintf("gauge_%d", i), float64(i))
		storage.UpdateCounter(fmt.Sprintf("counter_%d", i), int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetAllMetrics()
	}
}

func BenchmarkGetAllMetrics_Large(b *testing.B) {
	storage := NewMemStorage()

	for i := 0; i < 250; i++ {
		storage.UpdateGauge(fmt.Sprintf("gauge_%d", i), float64(i))
		storage.UpdateCounter(fmt.Sprintf("counter_%d", i), int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetAllMetrics()
	}
}

func BenchmarkUpdateBatch(b *testing.B) {
	storage := NewMemStorage()
	batch := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		v := float64(i)
		batch[i] = model.Metrics{
			ID:    fmt.Sprintf("metric_%d", i),
			MType: model.Gauge,
			Value: &v,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.UpdateBatch(batch)
	}
}

func BenchmarkUpdateBatch_Mixed(b *testing.B) {
	storage := NewMemStorage()
	batch := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			v := float64(i)
			batch[i] = model.Metrics{
				ID:    fmt.Sprintf("gauge_%d", i),
				MType: model.Gauge,
				Value: &v,
			}
		} else {
			d := int64(i)
			batch[i] = model.Metrics{
				ID:    fmt.Sprintf("counter_%d", i),
				MType: model.Counter,
				Delta: &d,
			}
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.UpdateBatch(batch)
	}
}

func BenchmarkConcurrentUpdates(b *testing.B) {
	storage := NewMemStorage()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			storage.UpdateGauge(fmt.Sprintf("gauge_%d", i%100), float64(i))
			storage.UpdateCounter(fmt.Sprintf("counter_%d", i%100), 1)
			i++
		}
	})
}
