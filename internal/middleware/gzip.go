// Package middleware содержит HTTP middleware для сервера метрик.
package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/SZabrodskii/go-metrics-stas/internal/pool"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gw          *gzip.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	w.ResponseWriter.Header().Set("Vary", "Accept-Encoding")
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.gw.Write(b)
}

func (w *gzipResponseWriter) Close() error {
	return w.gw.Close()
}

type contentWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	status     int
	headerSent bool
}

func (cw *contentWriter) Header() http.Header {
	return cw.ResponseWriter.Header()
}

func (cw *contentWriter) Write(b []byte) (int, error) {
	return cw.buf.Write(b)
}

func (cw *contentWriter) WriteHeader(statusCode int) {
	if cw.headerSent {
		return
	}
	cw.status = statusCode
	cw.headerSent = true
}

func (cw *contentWriter) statusOrDefault() int {
	if cw.status == 0 {
		return http.StatusOK
	}
	return cw.status
}

// CompressAccepted возвращает middleware для gzip сжатия ответов.
// Сжимает ответы только если клиент поддерживает gzip (Accept-Encoding: gzip)
// и Content-Type является сжимаемым (application/json, text/html).
func CompressAccepted(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		buf := pool.GetBuffer()
		defer pool.PutBuffer(buf)

		cw := &contentWriter{ResponseWriter: w, buf: buf}
		next.ServeHTTP(cw, r)

		ct := cw.Header().Get("Content-Type")
		if !isCompressible(ct) || cw.statusOrDefault() >= 300 {
			w.WriteHeader(cw.statusOrDefault())
			_, _ = w.Write(cw.buf.Bytes())
			return
		}

		zw := pool.GetGzipWriter(w)
		grw := &gzipResponseWriter{ResponseWriter: w, gw: zw}

		if !cw.headerSent {
			grw.WriteHeader(cw.statusOrDefault())
		}

		_, _ = grw.Write(cw.buf.Bytes())
		pool.PutGzipWriter(zw)
	})
}

func isCompressible(encoding string) bool {
	return strings.HasPrefix(encoding, "application/json") || strings.HasPrefix(encoding, "text/html")
}

// Decompress возвращает middleware для распаковки gzip-сжатых запросов.
// Распаковывает тело запроса если Content-Encoding: gzip.
func Decompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := r.Header.Get("Content-Encoding")
		if strings.EqualFold(enc, "gzip") && isCompressible(r.Header.Get("Content-Type")) {
			gr, err := pool.GetGzipReader(r.Body)
			if err != nil {
				http.Error(w, "unable to decompress gzipped body", http.StatusBadRequest)
				return
			}
			r.Body = struct {
				io.Reader
				io.Closer
			}{Reader: gr, Closer: pooledGzipReaderCloser{gr, r.Body}}
		}
		next.ServeHTTP(w, r)
	})
}

type pooledGzipReaderCloser struct {
	gr       *gzip.Reader
	original io.Closer
}

func (c pooledGzipReaderCloser) Close() error {
	pool.PutGzipReader(c.gr)
	return c.original.Close()
}

// CompressAndSign возвращает middleware для gzip сжатия и HMAC подписи ответов.
// Добавляет заголовок HashSHA256 с подписью тела ответа.
// Если key пустой, работает как CompressAccepted.
func CompressAndSign(key string, next http.Handler) http.Handler {
	if key == "" {
		return CompressAccepted(next)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := pool.GetBuffer()
		defer pool.PutBuffer(buf)

		cw := &contentWriter{ResponseWriter: w, buf: buf}
		next.ServeHTTP(cw, r)

		sum := hmacSHA256Hex(cw.buf.Bytes(), key)
		if sum != "" {
			cw.Header().Set("HashSHA256", sum)
		}

		ct := cw.Header().Get("Content-Type")
		if !isCompressible(ct) || cw.statusOrDefault() >= 300 || !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.WriteHeader(cw.statusOrDefault())
			_, _ = w.Write(cw.buf.Bytes())
			return
		}

		zw := pool.GetGzipWriter(w)
		grw := &gzipResponseWriter{ResponseWriter: w, gw: zw}

		if !cw.headerSent {
			grw.WriteHeader(cw.statusOrDefault())
		}
		_, _ = grw.Write(cw.buf.Bytes())
		pool.PutGzipWriter(zw)
	})
}
