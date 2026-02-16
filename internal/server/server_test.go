package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type noopShutdowner struct{}

func (noopShutdowner) Shutdown(...fx.ShutdownOption) error { return nil }

func TestLoadPrivateKey_EmptyPath(t *testing.T) {
	logger := zaptest.NewLogger(t)
	key, err := loadPrivateKey("", logger)
	assert.NoError(t, err)
	assert.Nil(t, key)
}

func TestLoadPrivateKey_InvalidPath(t *testing.T) {
	logger := zaptest.NewLogger(t)
	key, err := loadPrivateKey("/nonexistent/key", logger)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestNewRouter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{Key: ""}
	metricsHandler := handler.NewMetricsHandler(nil, zap.NewNop(), nil)
	pingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router, err := NewRouter(cfg, metricsHandler, pingHandler, logger)
	require.NoError(t, err)
	require.NotNil(t, router)

	expectedRoutes := map[string][]string{
		"GET":  {"/ping", "/"},
		"POST": {"/update", "/updates", "/value"},
	}
	routes := router.Routes()
	registeredPaths := make(map[string]map[string]bool)
	for _, route := range routes {
		collectRoutes(route, registeredPaths)
	}

	for method, paths := range expectedRoutes {
		for _, path := range paths {
			found, ok := registeredPaths[method]
			assert.True(t, ok && found[path], "route %s %s should be registered", method, path)
		}
	}
}

func collectRoutes(route chi.Route, result map[string]map[string]bool) {
	if route.SubRoutes != nil {
		for _, sub := range route.SubRoutes.Routes() {
			collectRoutes(sub, result)
		}
		return
	}

	for method := range route.Handlers {
		if _, ok := result[method]; !ok {
			result[method] = make(map[string]bool)
		}
		result[method][route.Pattern] = true
	}
}

func TestNewRouter_InvalidCryptoKey(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{CryptoKey: "/nonexistent/key.pem"}
	metricsHandler := handler.NewMetricsHandler(nil, zap.NewNop(), nil)
	pingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	router, err := NewRouter(cfg, metricsHandler, pingHandler, logger)
	assert.Error(t, err)
	assert.Nil(t, router)
}

func TestNewServer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{
		ListenAddress: "localhost:0",
		Key:           "",
	}
	router := chi.NewRouter()
	lc := fxtest.NewLifecycle(t)

	srv := NewServer(lc, router, cfg, logger, noopShutdowner{})
	require.NotNil(t, srv)
	assert.Equal(t, "localhost:0", srv.Addr)
	assert.NotNil(t, srv.Handler)
	assert.Equal(t, 5*time.Second, srv.ReadHeaderTimeout)
}
