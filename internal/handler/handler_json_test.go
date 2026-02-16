package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestHandler() *MetricsHandler {
	return NewMetricsHandler(repository.NewMemStorage(), zap.NewNop(), nil)
}

func setupRouter(h *MetricsHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetric)
	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/updates", h.UpdateBatchJSON)
	r.Get("/value/{type}/{name}", h.GetMetricValue)
	r.Post("/value", h.GetMetricValueJSON)
	r.Get("/", h.ListAllMetricsHTML)
	return r
}

func TestUpdateMetricJSON_Gauge(t *testing.T) {
	h := newTestHandler()
	val := 42.5
	body, _ := json.Marshal(map[string]interface{}{"id": "cpu", "type": "gauge", "value": val})

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "cpu", resp["id"])
	assert.Equal(t, "gauge", resp["type"])
	assert.InDelta(t, 42.5, resp["value"], 0.001)
}

func TestUpdateMetricJSON_Counter(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{"id": "hits", "type": "counter", "delta": 5})

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "hits", resp["id"])
	assert.Equal(t, "counter", resp["type"])
	assert.InDelta(t, 5, resp["delta"], 0.001)
}

func TestUpdateMetricJSON_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMetricJSON_MissingID(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{"type": "gauge", "value": 1.0})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMetricJSON_MissingType(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{"id": "test", "value": 1.0})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMetricJSON_InvalidType(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{"id": "test", "type": "unknown", "value": 1.0})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMetricJSON_GaugeNoValue(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{"id": "test", "type": "gauge"})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMetricJSON_CounterNoDelta(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]interface{}{"id": "test", "type": "counter"})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateMetricJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMetricValueJSON_Gauge(t *testing.T) {
	h := newTestHandler()
	h.repo.UpdateGauge("cpu", 75.5)

	body, _ := json.Marshal(map[string]string{"id": "cpu", "type": "gauge"})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.InDelta(t, 75.5, resp["value"], 0.001)
}

func TestGetMetricValueJSON_Counter(t *testing.T) {
	h := newTestHandler()
	h.repo.UpdateCounter("hits", 100)

	body, _ := json.Marshal(map[string]string{"id": "hits", "type": "counter"})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.InDelta(t, 100, resp["delta"], 0.001)
}

func TestGetMetricValueJSON_NotFound(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]string{"id": "unknown", "type": "gauge"})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMetricValueJSON_CounterNotFound(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]string{"id": "unknown", "type": "counter"})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMetricValueJSON_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMetricValueJSON_MissingFields(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]string{"id": "test"})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetMetricValueJSON_InvalidType(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal(map[string]string{"id": "test", "type": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.GetMetricValueJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBatchJSON_Valid(t *testing.T) {
	h := newTestHandler()
	val := 1.0
	delta := int64(5)
	batch := []model.Metrics{
		{ID: "g1", MType: "gauge", Value: &val},
		{ID: "c1", MType: "counter", Delta: &delta},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	g, err := h.repo.GetGauge("g1")
	require.NoError(t, err)
	assert.InDelta(t, 1.0, g, 0.001)

	c, err := h.repo.GetCounter("c1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), c)
}

func TestUpdateBatchJSON_EmptyBatch(t *testing.T) {
	h := newTestHandler()
	body, _ := json.Marshal([]model.Metrics{})
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `[]`, strings.TrimSpace(w.Body.String()))
}

func TestUpdateBatchJSON_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBatchJSON_MissingID(t *testing.T) {
	h := newTestHandler()
	val := 1.0
	batch := []model.Metrics{{MType: "gauge", Value: &val}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBatchJSON_InvalidType(t *testing.T) {
	h := newTestHandler()
	val := 1.0
	batch := []model.Metrics{{ID: "x", MType: "bad", Value: &val}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBatchJSON_GaugeNoValue(t *testing.T) {
	h := newTestHandler()
	batch := []model.Metrics{{ID: "x", MType: "gauge"}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateBatchJSON_CounterNoDelta(t *testing.T) {
	h := newTestHandler()
	batch := []model.Metrics{{ID: "x", MType: "counter"}}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateBatchJSON(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListAllMetricsHTML_Empty(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ListAllMetricsHTML(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "No metrics available")
}

func TestGetMetricValue_CounterNotFound(t *testing.T) {
	h := newTestHandler()

	r := setupRouter(h)
	req := httptest.NewRequest(http.MethodGet, "/value/counter/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMetricValue_InvalidGaugeValue(t *testing.T) {
	h := newTestHandler()
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test/notfloat", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
