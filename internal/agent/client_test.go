package agent

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHmacSHA256Hex_EmptyKey(t *testing.T) {
	result := hmacSHA256Hex([]byte("data"), "")
	assert.Equal(t, "", result)
}

func TestHmacSHA256Hex_NonEmptyKey(t *testing.T) {
	result := hmacSHA256Hex([]byte("hello"), "secret")
	assert.NotEmpty(t, result)
	assert.Len(t, result, 64)

	result2 := hmacSHA256Hex([]byte("hello"), "secret")
	assert.Equal(t, result, result2)

	result3 := hmacSHA256Hex([]byte("world"), "secret")
	assert.NotEqual(t, result, result3)
}

func TestShouldRetryHTTP_NetworkError(t *testing.T) {
	netErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	assert.True(t, shouldRetryHTTP(nil, netErr))
}

func TestShouldRetryHTTP_GenericError(t *testing.T) {
	assert.True(t, shouldRetryHTTP(nil, errors.New("something failed")))
}

func TestShouldRetryHTTP_500(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusInternalServerError}
	assert.True(t, shouldRetryHTTP(resp, nil))
}

func TestShouldRetryHTTP_200(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusOK}
	assert.False(t, shouldRetryHTTP(resp, nil))
}

func TestShouldRetryHTTP_502(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusBadGateway}
	assert.True(t, shouldRetryHTTP(resp, nil))
}

func TestShouldRetryHTTP_400(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusBadRequest}
	assert.False(t, shouldRetryHTTP(resp, nil))
}

func TestNewMetricsClient(t *testing.T) {
	mc := newMetricsClient("http://localhost:8080", "key", nil)
	require.NotNil(t, mc)
	assert.Equal(t, "http://localhost:8080", mc.serverURL)
	assert.Equal(t, "key", mc.key)
	assert.Nil(t, mc.publicKey)
	assert.NotNil(t, mc.client)
}

func TestSendBatch_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/updates", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mc := newMetricsClient(server.URL, "", nil)
	mc.client = &retryHTTPClient{
		base:          server.Client(),
		retrySchedule: nil,
	}

	val := 42.0
	batch := []model.Metrics{
		{ID: "test", MType: model.Gauge, Value: &val},
	}
	err := mc.SendBatch(batch)
	assert.NoError(t, err)
}

func TestSendBatch_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mc := newMetricsClient(server.URL, "", nil)
	mc.client = &retryHTTPClient{
		base:          server.Client(),
		retrySchedule: nil,
	}

	val := 1.0
	batch := []model.Metrics{
		{ID: "test", MType: model.Gauge, Value: &val},
	}
	err := mc.SendBatch(batch)
	assert.Error(t, err)
}

func TestSendBatch_EmptyBatch(t *testing.T) {
	mc := newMetricsClient("http://localhost:8080", "", nil)
	err := mc.SendBatch(nil)
	assert.NoError(t, err)

	err = mc.SendBatch([]model.Metrics{})
	assert.NoError(t, err)
}

func TestSendMetric_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/update", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	mc := newMetricsClient(server.URL, "", nil)
	mc.client = &retryHTTPClient{
		base:          server.Client(),
		retrySchedule: nil,
	}

	val := 99.9
	err := mc.SendMetric(model.Metrics{ID: "test", MType: model.Gauge, Value: &val})
	assert.NoError(t, err)
}

func TestSendBatch_WithHMACKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hash := r.Header.Get("HashSHA256")
		assert.NotEmpty(t, hash)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mc := newMetricsClient(server.URL, "mykey", nil)
	mc.client = &retryHTTPClient{
		base:          server.Client(),
		retrySchedule: nil,
	}

	val := 1.0
	batch := []model.Metrics{
		{ID: "test", MType: model.Gauge, Value: &val},
	}
	err := mc.SendBatch(batch)
	assert.NoError(t, err)
}
