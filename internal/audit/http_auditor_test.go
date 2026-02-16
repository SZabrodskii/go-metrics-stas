package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestHTTPAuditor_GetID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor("http://localhost:9999/audit", logger)
	assert.Equal(t, "http_auditor_http://localhost:9999/audit", ha.GetID())
}

func TestHTTPAuditor_Update_Success(t *testing.T) {
	var received AuditEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "metrics-audit-client/1.0", r.Header.Get("User-Agent"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor(srv.URL, logger)

	event := AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "10.0.0.1",
	}

	err := ha.Update(event)
	require.NoError(t, err)

	assert.Equal(t, event.Timestamp, received.Timestamp)
	assert.Equal(t, event.Metrics, received.Metrics)
	assert.Equal(t, event.IPAddress, received.IPAddress)
}

func TestHTTPAuditor_Update_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor(srv.URL, logger)

	err := ha.Update(AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"m"},
		IPAddress: "1.2.3.4",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestHTTPAuditor_Update_ConnectionRefused(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor("http://127.0.0.1:1", logger)

	err := ha.Update(AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"m"},
		IPAddress: "1.2.3.4",
	})
	assert.Error(t, err)
}

func TestHTTPAuditor_Update_BadURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor("://invalid", logger)

	err := ha.Update(AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"m"},
		IPAddress: "1.2.3.4",
	})
	assert.Error(t, err)
}

func TestHTTPAuditor_Update_Status299(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor(srv.URL, logger)

	err := ha.Update(AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"m"},
		IPAddress: "1.2.3.4",
	})
	assert.NoError(t, err)
}

func TestHTTPAuditor_Update_Status301(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer srv.Close()

	logger := zaptest.NewLogger(t)
	ha := NewHTTPAuditor(srv.URL, logger)

	err := ha.Update(AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"m"},
		IPAddress: "1.2.3.4",
	})
	assert.Error(t, err)
}
