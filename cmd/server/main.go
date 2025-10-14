package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/handler"
	mw "github.com/SZabrodskii/go-metrics-stas/internal/middleware"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/SZabrodskii/go-metrics-stas/pkg/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	addrFlag := flag.String("a", "localhost:8080", "HTTP listen address (host:port), e.g. localhost:8080")
	storeIntervalFlag := flag.Int("i", 300, "store interval in seconds (0 = sync write)")
	filePathFlag := flag.String("f", "metrics-db.json", "file to persist metrics")
	restoreFlag := flag.Bool("r", true, "restore metrics from file on start")
	flag.Parse()

	storeInterval := *storeIntervalFlag
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			storeInterval = n
		}
	}

	filePath := *filePathFlag
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		filePath = v
	}

	restore := *restoreFlag
	if v := os.Getenv("RESTORE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			restore = b
		}
	}

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

	if restore {
		if err = repository.LoadFromFile(storage, filePath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("error loading metrics from file %q: %v", filePath, err)
			}
		} else {
			log.Printf("loaded metrics from file %q", filePath)
		}

	}

	var onUpdate func()
	if storeInterval == 0 {
		onUpdate = func() {
			if err := repository.SaveToFile(storage, filePath); err != nil {
				log.Printf("error sync saving metrics to file %q: %v", filePath, err)
			}
		}
	}

	metricsHandler := handler.NewMetricsHandler(storage, onUpdate)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.StripSlashes, mw.Decompress, mw.ZapRequestLogger(logg), middleware.Recoverer)

	r.Post("/update/{type}/{name}/{value}", metricsHandler.UpdateMetric)
	r.Post("/update", metricsHandler.UpdateMetricJSON)
	r.Post("/value", metricsHandler.GetMetricValueJSON)
	r.Get("/value/{type}/{name}", metricsHandler.GetMetricValue)
	r.Get("/", metricsHandler.ListAllMetricsHTML)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mw.CompressAccepted(r),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("Starting server on port: ", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", srv.Addr, err)
		}
	}()

	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		go func() {
			for range ticker.C {
				if err := repository.SaveToFile(storage, filePath); err != nil {
					log.Printf("error periodic saving metrics to file %q: %v", filePath, err)
				}
			}
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	if ticker != nil {
		ticker.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown Failed:%+v", err)
	}
	if err = repository.SaveToFile(storage, filePath); err != nil {
		log.Printf("error final saving metrics to file %q: %v", filePath, err)
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
