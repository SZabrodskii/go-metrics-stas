package middleware

import (
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	rd *responseData
	http.ResponseWriter
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.rd.status = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.rd.size += n
	return n, err
}

// ZapRequestLogger возвращает middleware для логирования HTTP запросов.
// Логирует метод, URI, статус, длительность, размер ответа и remote_addr.
func ZapRequestLogger(log *zap.Logger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rd := &responseData{status: http.StatusOK}
			lrw := &loggingResponseWriter{ResponseWriter: w, rd: rd}

			next.ServeHTTP(lrw, r)

			log.Info("http request",
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.Int("status", rd.status),
				zap.Duration("duration", time.Since(start)),
				zap.Int("size", rd.size),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
