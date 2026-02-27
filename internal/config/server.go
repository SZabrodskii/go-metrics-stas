// Package config содержит конфигурацию сервера и агента.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/fx"
)

type serverJSONConfig struct {
	Address       string `json:"address"`
	Restore       *bool  `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	TrustedSubnet string `json:"trusted_subnet"`
	GRPCAddress   string `json:"grpc_address"`
}

// ServerConfig содержит конфигурацию HTTP сервера метрик.
type ServerConfig struct {
	// ListenAddress — адрес для прослушивания (host:port).
	ListenAddress string
	// StoreInterval — интервал сохранения метрик в файл (0 = синхронная запись).
	StoreInterval time.Duration
	// FileStoragePath — путь к файлу для персистентного хранения метрик.
	FileStoragePath string
	// Restore — восстанавливать ли метрики из файла при запуске.
	Restore bool
	// DatabaseDSN — строка подключения к PostgreSQL.
	DatabaseDSN string
	// Key — ключ для HMAC-SHA256 подписи (опционально).
	Key string
	// CryptoKey — путь к файлу приватного RSA-ключа для дешифрования (опционально).
	CryptoKey string
	// AuditFile — путь к файлу аудит-лога (опционально).
	AuditFile string
	// AuditURL — URL для удалённого аудит-лога (опционально).
	AuditURL      string
	TrustedSubnet string
	// GRPCAddress — адрес для прослушивания gRPC (host:port, опционально).
	GRPCAddress string
}

// NewServerConfig создаёт ServerConfig из флагов командной строки и переменных окружения.
// Переменные окружения имеют приоритет над флагами.
// Возвращает ошибку, если переменные окружения содержат некорректные значения.
func NewServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{}

	flag.StringVar(&cfg.ListenAddress, "a", "localhost:8080", "HTTP listen address (host:port)")
	storeIntervalSec := flag.Int("i", 300, "Store interval in seconds (0 = sync write)")
	flag.StringVar(&cfg.FileStoragePath, "f", "", "File to persist metrics (disabled by default)")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore metrics from file on start")

	flag.StringVar(&cfg.DatabaseDSN, "d", "", "PostgreSQL DSN (e.g. postgres://user:pass@host:5432/db?sslmode=disable)")
	flag.StringVar(&cfg.Key, "k", "", "Signing key for HMAC-SHA256 (optional)")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "Path to RSA private key PEM file for decryption (optional)")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "Path to audit log file (optional)")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "URL for remote audit logging (optional)")
	flag.StringVar(&cfg.TrustedSubnet, "t", "", "Trusted subnet in CIDR notation (optional)")
	flag.StringVar(&cfg.GRPCAddress, "grpc", "", "gRPC listen address (host:port, optional)")

	var configPath string

	flag.StringVar(&configPath, "c", "", "Path to JSON config file")
	flag.StringVar(&configPath, "config", "", "Path to JSON config file (same as -c)")

	flag.Parse()

	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
		}
		var jc serverJSONConfig

		if err := json.Unmarshal(data, &jc); err != nil {
			return nil, fmt.Errorf("parsing config file %s: %w", configPath, err)
		}

		if jc.Address != "" && !setFlags["a"] {
			cfg.ListenAddress = jc.Address
		}

		if jc.Restore != nil && !setFlags["r"] {
			cfg.Restore = *jc.Restore
		}

		if jc.StoreInterval != "" && !setFlags["i"] {
			d, err := time.ParseDuration(jc.StoreInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid store_interval in config file: %w", err)
			}
			*storeIntervalSec = int(d.Seconds())
		}

		if jc.StoreFile != "" && !setFlags["f"] {
			cfg.FileStoragePath = jc.StoreFile
		}

		if jc.DatabaseDSN != "" && !setFlags["d"] {
			cfg.DatabaseDSN = jc.DatabaseDSN
		}

		if jc.CryptoKey != "" && !setFlags["crypto-key"] {
			cfg.CryptoKey = jc.CryptoKey
		}
		if jc.TrustedSubnet != "" && !setFlags["t"] {
			cfg.TrustedSubnet = jc.TrustedSubnet
		}
		if jc.GRPCAddress != "" && !setFlags["grpc"] {
			cfg.GRPCAddress = jc.GRPCAddress
		}
	}

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		if v == "" {
			return nil, fmt.Errorf("ADDRESS is set but empty")
		}
		cfg.ListenAddress = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid STORE_INTERVAL %q: must be >= 0", v)
		}
		*storeIntervalSec = n
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		if v == "" {
			return nil, fmt.Errorf("FILE_STORAGE_PATH is set but empty")
		}
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		if v == "" {
			return nil, fmt.Errorf("RESTORE is set but empty")
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RESTORE %q: %w", v, err)
		}
		cfg.Restore = b
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = v
	}
	if k, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = k
	}
	if af, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = af
	}
	if au, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = au
	}
	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = v
	}
	if v, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		cfg.TrustedSubnet = v
	}
	if v, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		cfg.GRPCAddress = v
	}
	cfg.StoreInterval = time.Duration(*storeIntervalSec) * time.Second

	if strings.HasPrefix(cfg.ListenAddress, "http://") || strings.HasPrefix(cfg.ListenAddress, "https://") {
		return nil, fmt.Errorf("invalid value for LISTEN_ADDRESS URL: %v", cfg.ListenAddress)
	}
	return cfg, nil
}

// ProvideServerConfig возвращает fx.Option для внедрения ServerConfig.
func ProvideServerConfig() fx.Option {
	return fx.Provide(NewServerConfig)
}
