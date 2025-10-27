package server

import (
	"context"
	"net/http"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	mw "github.com/SZabrodskii/go-metrics-stas/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewRouter(metricsHandler *handler.MetricsHandler, pingHandler http.HandlerFunc, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.StripSlashes, mw.Decompress, mw.ZapRequestLogger(logger), middleware.Recoverer, middleware.RedirectSlashes)

	r.Get("/ping", pingHandler)

	r.Post("/update/{type}/{name}/{value}", metricsHandler.UpdateMetric)
	r.Post("/update", metricsHandler.UpdateMetricJSON)
	r.Post("/value", metricsHandler.GetMetricValueJSON)
	r.Get("/value/{type}/{name}", metricsHandler.GetMetricValue)
	r.Get("/", metricsHandler.ListAllMetricsHTML)
	r.Post("/updates", metricsHandler.UpdateBatchJSON)
	r.Post("/updates/", metricsHandler.UpdateBatchJSON)

	return r
}

func NewServer(lc fx.Lifecycle, router *chi.Mux, cfg *config.ServerConfig, logger *zap.Logger) *http.Server {
	srv := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mw.CompressAccepted(router),
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
