package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
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
	buf        bytes.Buffer
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

func CompressAccepted(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		cw := &contentWriter{ResponseWriter: w}
		next.ServeHTTP(cw, r)

		ct := cw.Header().Get("Content-Type")
		if !isCompressible(ct) || cw.statusOrDefault() >= 300 {
			if !cw.headerSent {
				w.WriteHeader(cw.statusOrDefault())
			}
			_, _ = w.Write(cw.buf.Bytes())
			return
		}

		grw := &gzipResponseWriter{ResponseWriter: w, gw: gzip.NewWriter(w)}
		defer grw.Close()

		if !cw.headerSent {
			grw.WriteHeader(cw.statusOrDefault())
		}

		_, _ = grw.Write(cw.buf.Bytes())
	})
}

func isCompressible(encoding string) bool {
	return strings.HasPrefix(encoding, "application/json") || strings.HasPrefix(encoding, "text/html")
}

func Decompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := r.Header.Get("Content-Encoding")
		if strings.EqualFold(enc, "gzip") && isCompressible(r.Header.Get("Content-Type")) {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "unable to decompress gzipped body", http.StatusBadRequest)
				return
			}
			r.Body = struct {
				io.Reader
				io.Closer
			}{Reader: gr, Closer: multiCloser{gr, r.Body}}
		}
		next.ServeHTTP(w, r)

	})
}

type multiCloser []io.Closer

func (mc multiCloser) Close() error {
	var firstErr error
	for _, c := range mc {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err

		}
	}
	return firstErr
}
