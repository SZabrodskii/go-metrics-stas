package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	logger, _ := zap.NewDevelopment()
	handler := NewMetricsHandler(storage, logger, nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetric)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid gauge update",
			method:         http.MethodPost,
			url:            "/update/gauge/testGauge/123.45",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Valid counter update",
			method:         http.MethodPost,
			url:            "/update/counter/testCounter/10",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Invalid http method",
			method:         http.MethodPut,
			url:            "/update/gauge/testGauge/123.45",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "Invalid metric type",
			method:         http.MethodPost,
			url:            "/update/invalid/test/123",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid metric type\n",
		},
		{
			name:           "Invalid metric format (not enough params)",
			method:         http.MethodPost,
			url:            "/update/gauge/",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
		},
		{
			name:           "Missing metric name (empty segment)",
			method:         http.MethodPost,
			url:            "/update/gauge//123.45",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "bad request: metric name is required\n",
		},
		{
			name:           "Invalid counter value",
			method:         http.MethodPost,
			url:            "/update/counter/c1/abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "bad request: counter must be int64\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "status code mismatch")

			if tt.expectedBody != "" {
				body := w.Body.String()
				assert.Equal(t, tt.expectedBody, body, "body mismatch")
			}
		})
	}
}

func TestGetMetricValue(t *testing.T) {
	storage := repository.NewMemStorage()
	logger, _ := zap.NewDevelopment()
	handler := NewMetricsHandler(storage, logger, nil)

	storage.UpdateGauge("testGauge", 123.45)
	storage.UpdateCounter("testCounter", 42)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", handler.GetMetricValue)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{
			name:           "Get existing gauge",
			method:         "GET",
			url:            "/value/gauge/testGauge",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get existing counter",
			method:         "GET",
			url:            "/value/counter/testCounter",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get non-existing metric",
			method:         "GET",
			url:            "/value/gauge/nonExisting",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid metric type",
			method:         "GET",
			url:            "/value/invalid/test",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestListAllMetricsHTML(t *testing.T) {
	storage := repository.NewMemStorage()
	logger, _ := zap.NewDevelopment()
	handler := NewMetricsHandler(storage, logger, nil)

	storage.UpdateGauge("testGauge", 123.45)
	storage.UpdateCounter("testCounter", 42)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ListAllMetricsHTML(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "testGauge") {
		t.Error("Expected HTML to contain testGauge")
	}
	if !strings.Contains(body, "testCounter") {
		t.Error("Expected HTML to contain testCounter")
	}
}
