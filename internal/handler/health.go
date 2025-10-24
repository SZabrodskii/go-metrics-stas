package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"go.uber.org/zap"
)

func NewPingHandler(db *sql.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			logger.Warn("ping: DB not configured (nil)")
			http.Error(w, "DB not configured", http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			logger.Error("ping: DB not reachable", zap.Error(err))
			http.Error(w, "db unreachable", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	}
}
