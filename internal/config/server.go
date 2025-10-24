package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/fx"
)

type ServerConfig struct {
	ListenAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
}

func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{}

	flag.StringVar(&cfg.ListenAddress, "a", "localhost:8080", "HTTP listen address (host:port)")
	storeIntervalSec := flag.Int("i", 300, "Store interval in seconds (0 = sync write)")
	flag.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "File to persist metrics")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore metrics from file on start")

	flag.StringVar(&cfg.DatabaseDSN, "d", "", "PostgreSQL DSN (e.g. postgres://user:pass@host:5432/db?sslmode=disable)")

	flag.Parse()

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		if v == "" {
			log.Fatalf("ADDRESS is set but empty")
		}
		cfg.ListenAddress = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			log.Fatalf("invalid STORE_INTERVAL %q: must be >= 0", v)
		}
		*storeIntervalSec = n
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		if v == "" {
			log.Fatalf("FILE_STORAGE_PATH is set but empty")
		}
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		if v == "" {
			log.Fatalf("RESTORE is set but empty")
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatalf("invalid RESTORE %q", v)
		}
		cfg.Restore = b
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = v
	}
	cfg.StoreInterval = time.Duration(*storeIntervalSec) * time.Second

	if strings.HasPrefix(cfg.ListenAddress, "http://") || strings.HasPrefix(cfg.ListenAddress, "https://") {
		log.Fatalf("invalid value for LISTEN_ADDRESS URL: %v", cfg.ListenAddress)
	}
	return cfg

}

func ProvideServerConfig() fx.Option {
	return fx.Provide(NewServerConfig)
}
