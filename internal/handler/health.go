package handler

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewPingHandler создаёт обработчик GET /ping для проверки соединения с БД.
// Возвращает 200 OK при успешном соединении, 500 при ошибке.
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

// RegisterSignalHandler регистрирует обработчик сигналов для graceful shutdown.
// Обрабатывает SIGTERM, SIGINT, SIGQUIT и инициирует завершение через fx.Shutdowner.
func RegisterSignalHandler(lc fx.Lifecycle, logger *zap.Logger, shutdowner fx.Shutdowner) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				sig := <-sigChan
				logger.Info("Received signal, initiating graceful shutdown", zap.String("signal", sig.String()))
				_ = shutdowner.Shutdown()
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			signal.Stop(sigChan)
			close(sigChan)
			return nil
		},
	})
}
