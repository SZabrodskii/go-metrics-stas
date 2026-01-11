package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"path"
	"strings"
)

func hmacSHA256Hex(b []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(b)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyHash возвращает middleware для проверки HMAC-SHA256 подписи запросов.
// Проверяет заголовок HashSHA256 для POST запросов на /update и /updates.
// Если key пустой, middleware пропускает все запросы без проверки.
func VerifyHash(key string) func(handler http.Handler) http.Handler {
	if key == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			p := path.Clean(r.URL.Path)
			if !strings.HasPrefix(p, "/update") || !strings.HasPrefix(p, "/updates") {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			provided := r.Header.Get("HashSHA256")
			if provided == "" {
				http.Error(w, "missing HashSHA256", http.StatusBadRequest)
				return
			}

			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "error found while reading request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(data))

			expected := hmacSHA256Hex(data, key)
			if !hmac.Equal([]byte(strings.ToLower(provided)), []byte(strings.ToLower(expected))) {
				http.Error(w, "bad signature", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
