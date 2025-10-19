package handler

import (
	"encoding/json"
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type MetricsHandler struct {
	repo   repository.Storage
	logger *zap.Logger
}

func NewMetricsHandler(repo repository.Storage, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		repo:   repo,
		logger: logger,
	}
}

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
		h.repo.UpdateGauge(metricName, value)
	case "counter":
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "bad request: counter must be int64", http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(metricName, delta)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

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
		value, err := h.repo.GetGauge(name)
		if err != nil {
			http.Error(w, "gauge not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatFloat(value, 'g', -1, 64)))

	case "counter":
		value, err := h.repo.GetCounter(name)
		if err != nil {
			http.Error(w, "counter not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatInt(value, 10)))
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
	}
}

func (h *MetricsHandler) ListAllMetricsHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	allMetrics, err := h.repo.GetAllMetrics()
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
		h.repo.UpdateGauge(m.ID, *m.Value)

		resp := struct {
			ID    string   `json:"id"`
			MType string   `json:"type"`
			Value *float64 `json:"value"`
		}{
			ID:    m.ID,
			MType: "gauge",
			Value: m.Value,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

	case "counter":
		if m.Delta == nil {
			http.Error(w, "delta is required for counter", http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(m.ID, *m.Delta)

		newVal, err := h.repo.GetCounter(m.ID)
		if err != nil {
			http.Error(w, "failed to retrieve updated counter", http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
}

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
		val, err := h.repo.GetGauge(q.ID)
		if err != nil {
			http.Error(w, "gauge not found", http.StatusNotFound)
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
		val, err := h.repo.GetCounter(q.ID)
		if err != nil {
			http.Error(w, "counter not found", http.StatusNotFound)
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
