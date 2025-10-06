package repository

import (
    "testing"
    "github.com/SZabrodskii/go-metrics-stas/internal/model"
	"sync"
)

func TestUpdateGauge(t *testing.T) {
	storage := NewMemStorage()

	storage.UpdateGauge("testGauge", 123.45)

	value, err := storage.GetGauge("testGauge")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != 123.45 {
		t.Errorf("expected 123.45, got %v", value)
	}

	storage.UpdateGauge("testGauge", 678.90)
	value, err = storage.GetGauge("testGauge")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != 678.90 {
		t.Errorf("expected 678.90, got %v", value)
	}
}

func TestUpdateCounter(t *testing.T) {
	storage := NewMemStorage()

	storage.UpdateCounter("testCounter", 10)

	value, err := storage.GetCounter("testCounter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != 10 {
		t.Errorf("expected 10, got %v", value)
	}

	storage.UpdateCounter("testCounter", 5)
	value, err = storage.GetCounter("testCounter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if value != 15 {
		t.Errorf("expected 15, got %v", value)
	}
}

func TestGetAllMetrics(t *testing.T) {
	storage := NewMemStorage()

	storage.UpdateGauge("testGauge", 123.45)
	storage.UpdateCounter("testCounter", 10)

	metrics, err := storage.GetAllMetrics()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if metrics["testGauge"].MType != model.Gauge {
		t.Errorf("expected testGauge to be a gauge, got %v", metrics["testGauge"].MType)
	}
	if metrics["testCounter"].MType != model.Counter {
		t.Errorf("expected testCounter to be a counter, got %v", metrics["testCounter"].MType)
	}
	if *metrics["testGauge"].Value != 123.45 {
		t.Errorf("expected testGauge value to be 123.45, got %v", metrics["testGauge"].Value)
	}
	if *metrics["testCounter"].Delta != 10 {
		t.Errorf("expected testCounter value to be 10, got %v", metrics["testCounter"].Delta)
	}
}

func TestThreadSafety(t *testing.T) {
	storage := NewMemStorage()

	numGoroutines := 100
	numOperations := 10

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				storage.UpdateCounter("concurrentCounter", 1)
			}
		} ()
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				storage.UpdateGauge("concurrentGauge", float64(j)*1.1)
			}
		} (float64(i))
	}


	wg.Wait()

	metrics, err := storage.GetAllMetrics()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedCounter := int64(numGoroutines * numOperations)
	if counter, exists := metrics["concurrentCounter"]; !exists {
		t.Error("expected concurrentCounter to exist")
	} else if *counter.Delta != expectedCounter {
		t.Errorf("expected testCounter delta to be %d, got %v", expectedCounter, *counter.Delta)
	}

	if _, exists := metrics["concurrentGauge"]; !exists {
		t.Errorf("expected concurrentGauge to be present")
	}
}

func TestCounterSummation(t *testing.T) {
	storage := NewMemStorage()

	storage.UpdateCounter("sumCounter", 5)
	storage.UpdateCounter("sumCounter", 10)
	storage.UpdateCounter("sumCounter", 3)

	metrics, err := storage.GetAllMetrics()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := int64(18)

	if counter, exists := metrics["sumCounter"]; !exists {
    t.Error("expected sumCounter to exist")
	} else if *counter.Delta != expected {
    t.Errorf("expected sumCounter delta to be %d, got %d", expected, *counter.Delta)
	}

	if *metrics["sumCounter"].Delta != expected {
		t.Errorf("expected sumCounter delta to be %d, got %v", expected, metrics["sumCounter"].Delta)
	}
}

func TestGaugeReplacement(t *testing.T) {
	storage := NewMemStorage()
	
	storage.UpdateGauge("replaceGauge", 100.5)
	storage.UpdateGauge("replaceGauge", 200.7)
	storage.UpdateGauge("replaceGauge", 50.3)

	metrics, err := storage.GetAllMetrics()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := 50.3
	if gauge, exists := metrics["replaceGauge"]; !exists {
    t.Error("expected replaceGauge to exist")
	} else if *gauge.Value != expected {
    t.Errorf("expected replaceGauge value to be %f, got %f", expected, *gauge.Value)
	}
	if *metrics["replaceGauge"].Value != expected {
		t.Errorf("expected replaceGauge value to be %f, got %v", expected, metrics["replaceGauge"].Value)
	}
}
