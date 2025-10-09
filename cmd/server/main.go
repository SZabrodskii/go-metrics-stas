package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	mw "github.com/SZabrodskii/go-metrics-stas/internal/middleware"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	addrFlag := flag.String("a", "localhost:8080", "HTTP listen address (host:port), e.g. localhost:8080")
	flag.Parse()

	addr := *addrFlag
	if v, ok := os.LookupEnv("ADDRESS"); ok && v != "" {
		addr = v
	}

	addr, err := normalizeListenAddress(addr)
	if err != nil {
		log.Fatalf("Invalid listen address: %v", err)
	}

	logg, cleanup := logging.NewLogger()
	defer cleanup()
	storage := repository.NewMemStorage()
	metricsHandler := handler.NewMetricsHandler(storage)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.StripSlashes, mw.ZapRequestLogger(logg), middleware.Recoverer)

	r.Post("/update/{type}/{name}/{value}", metricsHandler.UpdateMetric)
	r.Post("/update", metricsHandler.UpdateMetricJSON)
	r.Post("/value", metricsHandler.GetMetricValueJSON)
	r.Get("/value/{type}/{name}", metricsHandler.GetMetricValue)
	r.Get("/", metricsHandler.ListAllMetricsHTML)

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("Starting server on port: ", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", srv.Addr, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")

}
func normalizeListenAddress(addr string) (string, error) {
	if addr == "" {
		return "", fmt.Errorf("empty -a")
	}

	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		u, err := url.Parse(addr)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("invalid -a: %q: must be host:port", addr)
		}
		addr = u.Host
	}
	return addr, nil
}
