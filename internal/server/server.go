// Package server содержит HTTP сервер и маршрутизатор для API метрик.
package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	appcrypto "github.com/SZabrodskii/go-metrics-stas/internal/crypto"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	mw "github.com/SZabrodskii/go-metrics-stas/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewRouter создаёт и настраивает chi маршрутизатор со всеми эндпоинтами.
// Включает middleware для логирования, сжатия, подписи и pprof профилирование.
func NewRouter(cfg *config.ServerConfig, metricsHandler *handler.MetricsHandler, pingHandler http.HandlerFunc, logger *zap.Logger) (*chi.Mux, error) {
	privateKey, err := loadPrivateKey(cfg.CryptoKey, logger)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.StripSlashes, mw.Decrypt(privateKey), mw.Decompress, mw.VerifyHash(cfg.Key), mw.ZapRequestLogger(logger), middleware.Recoverer, middleware.RedirectSlashes)

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

	return r, nil
}

// NewServer создаёт HTTP сервер с fx lifecycle хуками для graceful shutdown.
func NewServer(lc fx.Lifecycle, router *chi.Mux, cfg *config.ServerConfig, logger *zap.Logger, shutdowner fx.Shutdowner) *http.Server {
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
					logger.Error("Error starting server", zap.Error(err))
					_ = shutdowner.Shutdown()
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

func loadPrivateKey(path string, logger *zap.Logger) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil
	}
	key, err := appcrypto.LoadPrivateKey(path)
	if err != nil {
		logger.Error("failed to load RSA private key", zap.String("path", path), zap.Error(err))
		return nil, fmt.Errorf("load RSA private key %s: %w", path, err)
	}
	logger.Info("RSA private key loaded for decryption", zap.String("path", path))
	return key, nil
}
