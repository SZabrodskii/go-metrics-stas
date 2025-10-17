package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	logger, _ := zap.NewDevelopment()
	testConfig := &config.ServerConfig{
		StoreInterval:   0 * time.Second,
		FileStoragePath: "/tmp/test-metrics-db.json",
		Restore:         false,
	}
	handler := NewMetricsHandler(storage, logger, testConfig)

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
