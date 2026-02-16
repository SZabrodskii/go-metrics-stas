package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCompressAccepted_WithGzip(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	h := CompressAccepted(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	gr, err := gzip.NewReader(w.Body)
	require.NoError(t, err)
	defer gr.Close()
	body, _ := io.ReadAll(gr)
	assert.Equal(t, `{"status":"ok"}`, string(body))
}

func TestCompressAccepted_NoGzip(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	h := CompressAccepted(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Equal(t, `{"status":"ok"}`, w.Body.String())
}

func TestCompressAccepted_NonCompressibleContentType(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("binary data"))
	})
	h := CompressAccepted(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Equal(t, "binary data", w.Body.String())
}

func TestCompressAccepted_ErrorStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"fail"}`))
	})
	h := CompressAccepted(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"error":"fail"}`, w.Body.String())
}

func TestDecompress_GzipBody(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"id":"test"}`))
	_ = zw.Close()

	var receivedBody string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusOK)
	})
	h := Decompress(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(buf.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"id":"test"}`, receivedBody)
}

func TestDecompress_NoGzip(t *testing.T) {
	var receivedBody string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusOK)
	})
	h := Decompress(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`plain`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "plain", receivedBody)
}

func TestDecompress_InvalidGzip(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Decompress(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("not gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDecompress_NonCompressibleType(t *testing.T) {
	var receivedBody string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusOK)
	})
	h := Decompress(inner)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("raw"))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "image/png")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "raw", receivedBody)
}

func TestCompressAndSign_EmptyKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	h := CompressAndSign("", inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("HashSHA256"))
}

func TestCompressAndSign_WithKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := CompressAndSign("secret", inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("HashSHA256"))
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
}

func TestCompressAndSign_WithKey_NoGzip(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := CompressAndSign("secret", inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("HashSHA256"))
	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Equal(t, `{"ok":true}`, w.Body.String())
}

func TestCompressAndSign_NonCompressible(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("img"))
	})
	h := CompressAndSign("key", inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, "img", w.Body.String())
	assert.NotEmpty(t, w.Header().Get("HashSHA256"))
}

func TestDecrypt_NilKey(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := Decrypt(nil)(inner)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDecrypt_GETRequest(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	key := generateTestRSAKey(t)
	h := Decrypt(key)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.True(t, called)
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	key := generateTestRSAKey(t)
	h := Decrypt(key)(inner)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not encrypted"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.True(t, called)
}

func TestZapRequestLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	})
	h := ZapRequestLogger(logger)(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "hello", w.Body.String())
}

func TestZapRequestLogger_DefaultStatus(t *testing.T) {
	logger := zaptest.NewLogger(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	h := ZapRequestLogger(logger)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIsCompressible(t *testing.T) {
	assert.True(t, isCompressible("application/json"))
	assert.True(t, isCompressible("application/json; charset=utf-8"))
	assert.True(t, isCompressible("text/html"))
	assert.True(t, isCompressible("text/html; charset=utf-8"))
	assert.False(t, isCompressible("image/png"))
	assert.False(t, isCompressible("text/plain"))
	assert.False(t, isCompressible(""))
}

func TestContentWriter_StatusOrDefault(t *testing.T) {
	cw := &contentWriter{buf: &bytes.Buffer{}}
	assert.Equal(t, http.StatusOK, cw.statusOrDefault())

	cw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, cw.statusOrDefault())

	cw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusNotFound, cw.statusOrDefault())
}

func TestGzipResponseWriter_DoubleWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	grw := &gzipResponseWriter{ResponseWriter: w, gw: zw}

	grw.WriteHeader(http.StatusOK)
	grw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusOK, w.Code)
}
