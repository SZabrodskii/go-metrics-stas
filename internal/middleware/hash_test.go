package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHmacSHA256Hex(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		key  string
	}{
		{
			name: "Basic test",
			data: []byte("test data"),
			key:  "secret",
		},
		{
			name: "Empty data",
			data: []byte(""),
			key:  "secret",
		},
		{
			name: "Empty key",
			data: []byte("test"),
			key:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hmacSHA256Hex(tt.data, tt.key)
			if len(result) == 0 {
				t.Error("Expected non-empty hash")
			}
			if len(result) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(result))
			}
		})
	}
}

func TestVerifyHash(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		method       string
		url          string
		contentType  string
		body         string
		hashHeader   string
		expectStatus int
	}{
		{
			name:         "No key - pass through",
			key:          "",
			method:       "POST",
			url:          "/update",
			expectStatus: http.StatusOK,
		},
		{
			name:         "GET request - pass through",
			key:          "secret",
			method:       "GET",
			url:          "/update",
			expectStatus: http.StatusOK,
		},
		{
			name:         "Valid hash",
			key:          "secret",
			method:       "POST",
			url:          "/update",
			contentType:  "application/json",
			body:         `{"test":"data"}`,
			hashHeader:   hmacSHA256Hex([]byte(`{"test":"data"}`), "secret"),
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := VerifyHash(tt.key)
			handler := middleware(testHandler)

			req := httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.hashHeader != "" {
				req.Header.Set("HashSHA256", tt.hashHeader)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}
		})
	}
}
