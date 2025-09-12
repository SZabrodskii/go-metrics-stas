package handler

import (
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"net/http"
	"net/url"
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
		http.Error(w, "path format is invalid", http.StatusNotFound)
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
		http.Error(w, "not found: metric name", http.StatusNotFound)
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
