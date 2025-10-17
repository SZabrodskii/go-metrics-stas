package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type ServerConfig struct {
	ListenAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
}

func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{}

	flag.StringVar(&cfg.ListenAddress, "a", "localhost:8080", "HTTP listen address (host:port)")
	storeIntervalSec := flag.Int("i", 300, "Store interval in seconds (0 = sync write)")
	flag.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "File to persist metrics")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore metrics from file on start")

	flag.Parse()

	if addr, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ListenAddress = addr
	}
	if val, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		n, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("invalid value for STORE_INTERVAL %q: %v", val, err)
		}
		*storeIntervalSec = n
	}
	if path, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = path
	}
	if val, ok := os.LookupEnv("RESTORE"); ok {
		b, err := strconv.ParseBool(val)
		if err != nil {
			log.Fatalf("invalid value for RESTORE %q: %v", val, err)
		}
		cfg.Restore = b
	}
	cfg.StoreInterval = time.Duration(*storeIntervalSec) * time.Second

	if strings.HasPrefix(cfg.ListenAddress, "http://") || strings.HasPrefix(cfg.ListenAddress, "https://") {
		log.Fatalf("invalid value for LISTEN_ADDRESS URL: %v", cfg.ListenAddress)
	}
	return cfg

}
