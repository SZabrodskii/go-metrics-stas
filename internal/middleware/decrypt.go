package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	appcrypto "github.com/SZabrodskii/go-metrics-stas/internal/crypto"
)

func Decrypt(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if privateKey == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			plaintext, err := appcrypto.Decrypt(body, privateKey)
			if err != nil {
				r.Body = io.NopCloser(bytes.NewReader(body))
				next.ServeHTTP(w, r)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))
			next.ServeHTTP(w, r)
		})
	}
}
