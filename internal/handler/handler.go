// Package handler содержит HTTP обработчики для API метрик.
package handler

import (
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/SZabrodskii/go-metrics-stas/internal/audit"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/SZabrodskii/go-metrics-stas/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// MetricsHandler обрабатывает HTTP запросы для работы с метриками.
// Предоставляет эндпоинты для обновления и получения метрик.
type MetricsHandler struct {
	svc       service.MetricsService
	logger    *zap.Logger
	publisher *audit.Publisher
}

// NewMetricsHandler создаёт новый экземпляр MetricsHandler.
// Принимает сервис метрик, логгер и опциональный publisher для аудита.
func NewMetricsHandler(svc service.MetricsService, logger *zap.Logger, publisher *audit.Publisher) *MetricsHandler {
	return &MetricsHandler{
		svc:       svc,
		logger:    logger,
		publisher: publisher,
	}
}

func (h *MetricsHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidMetricType):
		http.Error(w, "invalid metric type", http.StatusBadRequest)
	case errors.Is(err, service.ErrMetricNotFound):
		http.Error(w, "metric not found", http.StatusNotFound)
	case errors.Is(err, service.ErrInvalidMetricID):
		http.Error(w, "id and type are required", http.StatusBadRequest)
	case errors.Is(err, service.ErrMissingValue):
		http.Error(w, "value is required for gauge", http.StatusBadRequest)
	case errors.Is(err, service.ErrMissingDelta):
		http.Error(w, "delta is required for counter", http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// UpdateMetric обрабатывает POST /update/{type}/{name}/{value}.
// Обновляет метрику, переданную через URL параметры.
func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	if metricName == "" {
		http.Error(w, "bad request: metric name is required", http.StatusBadRequest)
		return
	}
	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "bad request: gauge must be float64", http.StatusBadRequest)
			return
		}
		if err := h.svc.UpdateGauge(metricName, value); err != nil {
			h.handleServiceError(w, err)
			return
		}
	case "counter":
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "bad request: counter must be int64", http.StatusBadRequest)
			return
		}
		if _, err := h.svc.UpdateCounter(metricName, delta); err != nil {
			h.handleServiceError(w, err)
			return
		}
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	if h.publisher != nil {
		event := audit.CreateAuditEvent(r, []string{metricName})
		h.publisher.Notify(event)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GetMetricValue обрабатывает GET /value/{type}/{name}.
// Возвращает значение метрики в текстовом формате.
func (h *MetricsHandler) GetMetricValue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	tp := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	if tp == "" || name == "" {
		http.Error(w, "path format is invalid", http.StatusBadRequest)
		return
	}

	switch tp {
	case "gauge":
		value, err := h.svc.GetGauge(name)
		if err != nil {
			if errors.Is(err, service.ErrMetricNotFound) {
				http.Error(w, "gauge not found", http.StatusNotFound)
				return
			}
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatFloat(value, 'g', -1, 64)))

	case "counter":
		value, err := h.svc.GetCounter(name)
		if err != nil {
			if errors.Is(err, service.ErrMetricNotFound) {
				http.Error(w, "counter not found", http.StatusNotFound)
				return
			}
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatInt(value, 10)))
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
	}
}

// ListAllMetricsHTML обрабатывает GET /.
// Возвращает HTML страницу со списком всех метрик.
func (h *MetricsHandler) ListAllMetricsHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	allMetrics, err := h.svc.GetAllMetrics()
	if err != nil {
		http.Error(w, "failed to retrieve metrics", http.StatusInternalServerError)
		return
	}

	names := make([]string, 0, len(allMetrics))
	for n := range allMetrics {
		names = append(names, n)
	}

	sort.Strings(names)

	var builder strings.Builder
	builder.WriteString("<!doctype html><html><head><meta charset='utf-8'><title>Metrics</title>")
	builder.WriteString("<style>body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Arial}table{border-collapse:collapse}td,th{border:1px solid #ddd;padding:6px 10px}</style>")
	builder.WriteString("</head><body><h1>Metrics</h1>")

	if len(names) == 0 {
		builder.WriteString("<p>No metrics available</p>")
	} else {
		builder.WriteString("<table><thead><tr><th>Name</th><th>Type</th><th>Value</th></tr></thead><tbody>")
		for _, n := range names {
			m := allMetrics[n]
			var valueStr string
			if m.Value != nil {
				valueStr = strconv.FormatFloat(*m.Value, 'g', -1, 64)
			}
			if m.Delta != nil {
				valueStr = strconv.FormatInt(*m.Delta, 10)
			}
			builder.WriteString("<tr><td>")
			builder.WriteString(html.EscapeString(n))
			builder.WriteString("</td><td>")
			builder.WriteString(html.EscapeString(m.MType))
			builder.WriteString("</td><td>")
			builder.WriteString(html.EscapeString(valueStr))
			builder.WriteString("</td></tr>")
		}
		builder.WriteString("</tbody></table>")
	}
	builder.WriteString("</body></html>")
	_, _ = w.Write([]byte(builder.String()))
}

// UpdateMetricJSON обрабатывает POST /update.
// Обновляет метрику, переданную в JSON формате.
// Возвращает обновлённую метрику в JSON.
func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var m struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Delta *int64   `json:"delta,omitempty"`
		Value *float64 `json:"value,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if m.ID == "" || m.MType == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}
	switch m.MType {
	case "gauge":
		if m.Value == nil {
			http.Error(w, "value is required for gauge", http.StatusBadRequest)
			return
		}
		if err := h.svc.UpdateGauge(m.ID, *m.Value); err != nil {
			h.handleServiceError(w, err)
			return
		}

		resp := struct {
			ID    string   `json:"id"`
			MType string   `json:"type"`
			Value *float64 `json:"value"`
		}{
			ID:    m.ID,
			MType: "gauge",
			Value: m.Value,
		}

		if h.publisher != nil {
			event := audit.CreateAuditEvent(r, []string{m.ID})
			h.publisher.Notify(event)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

	case "counter":
		if m.Delta == nil {
			http.Error(w, "delta is required for counter", http.StatusBadRequest)
			return
		}
		newVal, err := h.svc.UpdateCounter(m.ID, *m.Delta)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}

		resp := struct {
			ID    string `json:"id"`
			MType string `json:"type"`
			Delta *int64 `json:"delta"`
		}{
			ID:    m.ID,
			MType: "counter",
			Delta: &newVal,
		}

		if h.publisher != nil {
			event := audit.CreateAuditEvent(r, []string{m.ID})
			h.publisher.Notify(event)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

// GetMetricValueJSON обрабатывает POST /value.
// Возвращает значение метрики в JSON формате.
func (h *MetricsHandler) GetMetricValueJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var q struct {
		ID    string `json:"id"`
		MType string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if q.ID == "" || q.MType == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}

	switch q.MType {
	case "gauge":
		val, err := h.svc.GetGauge(q.ID)
		if err != nil {
			if errors.Is(err, service.ErrMetricNotFound) {
				http.Error(w, "gauge not found", http.StatusNotFound)
				return
			}
			h.handleServiceError(w, err)
			return
		}
		resp := struct {
			ID    string   `json:"id"`
			MType string   `json:"type"`
			Value *float64 `json:"value"`
		}{
			ID:    q.ID,
			MType: "gauge",
			Value: &val,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

	case "counter":
		val, err := h.svc.GetCounter(q.ID)
		if err != nil {
			if errors.Is(err, service.ErrMetricNotFound) {
				http.Error(w, "counter not found", http.StatusNotFound)
				return
			}
			h.handleServiceError(w, err)
			return
		}
		resp := struct {
			ID    string `json:"id"`
			MType string `json:"type"`
			Delta *int64 `json:"delta"`
		}{
			ID:    q.ID,
			MType: "counter",
			Delta: &val,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

}

// UpdateBatchJSON обрабатывает POST /updates.
// Обновляет несколько метрик за один запрос (batch update).
// Принимает массив метрик в JSON формате.
func (h *MetricsHandler) UpdateBatchJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	defer r.Body.Close()

	var batch []model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if len(batch) == 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
		return
	}

	if err := h.svc.UpdateBatch(batch); err != nil {
		h.handleServiceError(w, err)
		return
	}

	if h.publisher != nil {
		metricNames := make([]string, 0, len(batch))
		for _, m := range batch {
			metricNames = append(metricNames, m.ID)
		}
		event := audit.CreateAuditEvent(r, metricNames)
		h.publisher.Notify(event)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(batch)
}
