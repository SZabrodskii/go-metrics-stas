package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func setupTestServer() *httptest.Server {
	storage := repository.NewMemStorage()
	logger := zap.NewNop()
	h := handler.NewMetricsHandler(storage, logger, nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Get("/value/{type}/{name}", h.GetMetricValue)
	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/update/", h.UpdateMetricJSON)
	r.Post("/value", h.GetMetricValueJSON)
	r.Post("/value/", h.GetMetricValueJSON)
	r.Post("/updates", h.UpdateBatchJSON)
	r.Post("/updates/", h.UpdateBatchJSON)
	r.Get("/", h.ListAllMetricsHTML)

	return httptest.NewServer(r)
}

// Example_updateMetricURL demonstrates updating a metric via URL parameters.
// POST /update/{type}/{name}/{value}
func Example_updateMetricURL() {
	ts := setupTestServer()
	defer ts.Close()

	// Update gauge metric
	resp, _ := http.Post(ts.URL+"/update/gauge/temperature/36.6", "text/plain", nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Body: %s\n", body)

	// Output:
	// Status: 200
	// Body: OK
}

// Example_getMetricValueURL demonstrates getting a metric value via URL.
// GET /value/{type}/{name}
func Example_getMetricValueURL() {
	ts := setupTestServer()
	defer ts.Close()

	// First, set a gauge value
	http.Post(ts.URL+"/update/gauge/cpu/75.5", "text/plain", nil)

	// Then retrieve it
	resp, _ := http.Get(ts.URL + "/value/gauge/cpu")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Value: %s\n", body)

	// Output:
	// Status: 200
	// Value: 75.5
}

// Example_updateMetricJSON demonstrates updating a metric via JSON body.
// POST /update with JSON payload
func Example_updateMetricJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// Update gauge metric via JSON
	value := 123.45
	payload := map[string]interface{}{
		"id":    "memory",
		"type":  "gauge",
		"value": value,
	}
	jsonData, _ := json.Marshal(payload)

	resp, _ := http.Post(ts.URL+"/update", "application/json", bytes.NewBuffer(jsonData))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("ID: %s\n", result["id"])
	fmt.Printf("Type: %s\n", result["type"])

	// Output:
	// Status: 200
	// ID: memory
	// Type: gauge
}

// Example_updateCounterJSON demonstrates updating a counter metric via JSON.
// POST /update with counter delta
func Example_updateCounterJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// Update counter metric via JSON
	delta := int64(5)
	payload := map[string]interface{}{
		"id":    "requests",
		"type":  "counter",
		"delta": delta,
	}
	jsonData, _ := json.Marshal(payload)

	resp, _ := http.Post(ts.URL+"/update", "application/json", bytes.NewBuffer(jsonData))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("ID: %s\n", result["id"])
	fmt.Printf("Type: %s\n", result["type"])
	fmt.Printf("Delta: %.0f\n", result["delta"])

	// Output:
	// Status: 200
	// ID: requests
	// Type: counter
	// Delta: 5
}

// Example_getMetricValueJSON demonstrates getting a metric value via JSON.
// POST /value with JSON query
func Example_getMetricValueJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// First, set a gauge value
	value := 99.9
	updatePayload := map[string]interface{}{
		"id":    "disk",
		"type":  "gauge",
		"value": value,
	}
	jsonData, _ := json.Marshal(updatePayload)
	http.Post(ts.URL+"/update", "application/json", bytes.NewBuffer(jsonData))

	// Then query it via JSON
	queryPayload := map[string]string{
		"id":   "disk",
		"type": "gauge",
	}
	jsonData, _ = json.Marshal(queryPayload)

	resp, _ := http.Post(ts.URL+"/value", "application/json", bytes.NewBuffer(jsonData))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("ID: %s\n", result["id"])
	fmt.Printf("Value: %.1f\n", result["value"])

	// Output:
	// Status: 200
	// ID: disk
	// Value: 99.9
}

// Example_updateBatchJSON demonstrates batch updating multiple metrics.
// POST /updates with array of metrics
func Example_updateBatchJSON() {
	ts := setupTestServer()
	defer ts.Close()

	// Update multiple metrics at once
	gaugeVal := 42.0
	counterDelta := int64(10)
	batch := []map[string]interface{}{
		{"id": "gauge1", "type": "gauge", "value": gaugeVal},
		{"id": "counter1", "type": "counter", "delta": counterDelta},
	}
	jsonData, _ := json.Marshal(batch)

	resp, _ := http.Post(ts.URL+"/updates", "application/json", bytes.NewBuffer(jsonData))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result []map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Metrics count: %d\n", len(result))

	// Output:
	// Status: 200
	// Metrics count: 2
}

// Example_listAllMetricsHTML demonstrates listing all metrics as HTML.
// GET /
func Example_listAllMetricsHTML() {
	ts := setupTestServer()
	defer ts.Close()

	// First, add some metrics
	http.Post(ts.URL+"/update/gauge/temp/25", "text/plain", nil)
	http.Post(ts.URL+"/update/counter/hits/1", "text/plain", nil)

	// Get HTML listing
	resp, _ := http.Get(ts.URL + "/")
	resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))

	// Output:
	// Status: 200
	// Content-Type: text/html; charset=utf-8
}
