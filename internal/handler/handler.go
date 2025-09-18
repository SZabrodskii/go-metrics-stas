package handler

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type MetricsHandler struct {
	repo repository.Storage
}

func NewMetricsHandler(repo repository.Storage) *MetricsHandler {
	return &MetricsHandler{repo: repo}
}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const prefix = "/update/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] == "" {
		http.Error(w, "path format is invalid", http.StatusBadRequest)
		return
	}

	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "bad request: metric value is required", http.StatusBadRequest)
		return
	}

	if len(parts) > 3 {
		http.Error(w, "bad request: too many parameters", http.StatusBadRequest)
		return
	}

	metricType := parts[0]
	metricName, err := url.PathUnescape(parts[1])
	if err != nil || metricName == "" {
		http.Error(w, "not found: metric name", http.StatusBadRequest)
		return
	}
	metricValue := parts[2]

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "bad request: gauge must be float64", http.StatusBadRequest)
			return
		}
		h.repo.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	case "counter":
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "bad request: counter must be int64", http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(metricName, delta)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}
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
