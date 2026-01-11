// Package server содержит HTTP сервер и маршрутизатор для API метрик.
package server

import (
	"context"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	mw "github.com/SZabrodskii/go-metrics-stas/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewRouter создаёт и настраивает chi маршрутизатор со всеми эндпоинтами.
// Включает middleware для логирования, сжатия, подписи и pprof профилирование.
func NewRouter(cfg *config.ServerConfig, metricsHandler *handler.MetricsHandler, pingHandler http.HandlerFunc, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.StripSlashes, mw.Decompress, mw.VerifyHash(cfg.Key), mw.ZapRequestLogger(logger), middleware.Recoverer, middleware.RedirectSlashes)

	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))
	r.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

	r.Get("/ping", pingHandler)

	r.Post("/update/{type}/{name}/{value}", metricsHandler.UpdateMetric)
	r.Post("/update", metricsHandler.UpdateMetricJSON)
	r.Post("/update/", metricsHandler.UpdateMetricJSON)
	r.Post("/value", metricsHandler.GetMetricValueJSON)
	r.Post("/value/", metricsHandler.GetMetricValueJSON)
	r.Get("/value/{type}/{name}", metricsHandler.GetMetricValue)
	r.Get("/", metricsHandler.ListAllMetricsHTML)
	r.Post("/updates", metricsHandler.UpdateBatchJSON)
	r.Post("/updates/", metricsHandler.UpdateBatchJSON)

	return r
}

// NewServer создаёт HTTP сервер с fx lifecycle хуками для graceful shutdown.
func NewServer(lc fx.Lifecycle, router *chi.Mux, cfg *config.ServerConfig, logger *zap.Logger) *http.Server {
	srv := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mw.CompressAndSign(cfg.Key, router),
		ReadHeaderTimeout: 5 * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting server", zap.String("address", srv.Addr))
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("Error starting server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting server down", zap.String("address", srv.Addr))
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	})
	return srv
}
