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

func Example_updateMetricURL() {
	ts := setupTestServer()
	defer ts.Close()

	resp, _ := http.Post(ts.URL+"/update/gauge/temperature/36.6", "text/plain", nil)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Body: %s\n", body)
}

func Example_getMetricValueURL() {
	ts := setupTestServer()
	defer ts.Close()

	setupResp, _ := http.Post(ts.URL+"/update/gauge/cpu/75.5", "text/plain", nil)
	setupResp.Body.Close()

	resp, _ := http.Get(ts.URL + "/value/gauge/cpu")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Value: %s\n", body)
}

func Example_updateMetricJSON() {
	ts := setupTestServer()
	defer ts.Close()

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
}

func Example_updateCounterJSON() {
	ts := setupTestServer()
	defer ts.Close()

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
}

func Example_getMetricValueJSON() {
	ts := setupTestServer()
	defer ts.Close()

	value := 99.9
	updatePayload := map[string]interface{}{
		"id":    "disk",
		"type":  "gauge",
		"value": value,
	}
	jsonData, _ := json.Marshal(updatePayload)
	setupResp, _ := http.Post(ts.URL+"/update", "application/json", bytes.NewBuffer(jsonData))
	setupResp.Body.Close()

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
}

func Example_updateBatchJSON() {
	ts := setupTestServer()
	defer ts.Close()

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
}

func Example_listAllMetricsHTML() {
	ts := setupTestServer()
	defer ts.Close()

	r1, _ := http.Post(ts.URL+"/update/gauge/temp/25", "text/plain", nil)
	r1.Body.Close()
	r2, _ := http.Post(ts.URL+"/update/counter/hits/1", "text/plain", nil)
	r2.Body.Close()

	resp, _ := http.Get(ts.URL + "/")
	resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
}
