package middleware

import (
	"errors"
	"net"
	"net/http"
)

var ErrInvalidCIDR = errors.New("invalid trusted_subnet CIDR")

func CheckTrustedSubnet(cidr string) (func(http.Handler) http.Handler, error) {
	if cidr == "" {
		return func(next http.Handler) http.Handler { return next }, nil
	}

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, errors.Join(ErrInvalidCIDR, err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get("X-Real-IP")
			if ipStr == "" {
				http.Error(w, "missing X-Real-IP header", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(ipStr)
			if ip == nil {
				http.Error(w, "invalid X-Real-IP header", http.StatusForbidden)
				return
			}

			if !subnet.Contains(ip) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}
