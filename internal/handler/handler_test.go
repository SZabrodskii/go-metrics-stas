package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
)

func TestUpdateMetric(t *testing.T) {
	storage := repository.NewMemStorage()
	handler := NewMetricsHandler(storage)

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
			expectedBody:   "method not allowed\n",
		},
		{
			name:           "Invalid metric type",
			method:         http.MethodPost,
			url:            "/update/invalid/test/123",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid metric type\n",
		},
		{
			name:           "Invalid metric format",
			method:         http.MethodPost,
			url:            "/update/gauge/",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "path format is invalid\n",
		},
		{
			name:           "Missing metric name",
			method:         http.MethodPost,
			url:            "/update/gauge//123.45",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "path format is invalid\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			handler.UpdateMetric(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			body := w.Body.String()
			if body != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, body)
			}
			defer resp.Body.Close()
		})
	}
}
