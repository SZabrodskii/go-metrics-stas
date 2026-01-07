package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func setupBenchmarkHandler() (*MetricsHandler, *chi.Mux) {
	storage := repository.NewMemStorage()
	logger, _ := zap.NewDevelopment()
	handler := NewMetricsHandler(storage, logger, nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetric)
	r.Post("/update", handler.UpdateMetricJSON)
	r.Post("/updates", handler.UpdateBatchJSON)
	r.Get("/value/{type}/{name}", handler.GetMetricValue)
	r.Post("/value", handler.GetMetricValueJSON)
	r.Get("/", handler.ListAllMetricsHTML)

	return handler, r
}

func BenchmarkUpdateMetric(b *testing.B) {
	_, r := setupBenchmarkHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/update/gauge/test_%d/%d.5", i%100, i), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateMetricJSON(b *testing.B) {
	_, r := setupBenchmarkHandler()

	value := 123.45
	payload := struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Value *float64 `json:"value"`
	}{
		ID:    "testGauge",
		MType: "gauge",
		Value: &value,
	}
	body, _ := json.Marshal(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateBatchJSON_Small(b *testing.B) {
	_, r := setupBenchmarkHandler()

	batch := make([]model.Metrics, 10)
	for i := 0; i < 10; i++ {
		v := float64(i)
		batch[i] = model.Metrics{
			ID:    fmt.Sprintf("metric_%d", i),
			MType: "gauge",
			Value: &v,
		}
	}
	body, _ := json.Marshal(batch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateBatchJSON_Medium(b *testing.B) {
	_, r := setupBenchmarkHandler()

	batch := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		v := float64(i)
		batch[i] = model.Metrics{
			ID:    fmt.Sprintf("metric_%d", i),
			MType: "gauge",
			Value: &v,
		}
	}
	body, _ := json.Marshal(batch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateBatchJSON_Large(b *testing.B) {
	_, r := setupBenchmarkHandler()

	batch := make([]model.Metrics, 500)
	for i := 0; i < 500; i++ {
		if i%2 == 0 {
			v := float64(i)
			batch[i] = model.Metrics{
				ID:    fmt.Sprintf("gauge_%d", i),
				MType: "gauge",
				Value: &v,
			}
		} else {
			d := int64(i)
			batch[i] = model.Metrics{
				ID:    fmt.Sprintf("counter_%d", i),
				MType: "counter",
				Delta: &d,
			}
		}
	}
	body, _ := json.Marshal(batch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkGetMetricValue(b *testing.B) {
	handler, r := setupBenchmarkHandler()
	handler.repo.UpdateGauge("testGauge", 123.45)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/value/gauge/testGauge", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkGetMetricValueJSON(b *testing.B) {
	handler, r := setupBenchmarkHandler()
	handler.repo.UpdateGauge("testGauge", 123.45)

	payload := struct {
		ID    string `json:"id"`
		MType string `json:"type"`
	}{
		ID:    "testGauge",
		MType: "gauge",
	}
	body, _ := json.Marshal(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkListAllMetricsHTML_Small(b *testing.B) {
	handler, r := setupBenchmarkHandler()

	for i := 0; i < 5; i++ {
		handler.repo.UpdateGauge(fmt.Sprintf("gauge_%d", i), float64(i))
		handler.repo.UpdateCounter(fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkListAllMetricsHTML_Medium(b *testing.B) {
	handler, r := setupBenchmarkHandler()

	for i := 0; i < 50; i++ {
		handler.repo.UpdateGauge(fmt.Sprintf("gauge_%d", i), float64(i))
		handler.repo.UpdateCounter(fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkListAllMetricsHTML_Large(b *testing.B) {
	handler, r := setupBenchmarkHandler()

	for i := 0; i < 250; i++ {
		handler.repo.UpdateGauge(fmt.Sprintf("gauge_%d", i), float64(i))
		handler.repo.UpdateCounter(fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
