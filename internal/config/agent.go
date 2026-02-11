package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/fx"
)

// AgentConfig содержит конфигурацию агента сбора метрик.
type AgentConfig struct {
	// PollInterval — интервал сбора метрик.
	PollInterval time.Duration
	// ReportInterval — интервал отправки метрик на сервер.
	ReportInterval time.Duration
	// ServerAddress — адрес сервера метрик (http://host:port).
	ServerAddress string
	// Key — ключ для HMAC-SHA256 подписи (опционально).
	Key string
	// RateLimit — количество параллельных запросов (0 = без ограничений).
	RateLimit int
	// CryptoKey — путь к файлу публичного RSA-ключа для шифрования (опционально).
	CryptoKey string
}

// NewAgentConfig создаёт AgentConfig из флагов командной строки и переменных окружения.
// Переменные окружения имеют приоритет над флагами.
// Возвращает ошибку, если переменные окружения содержат некорректные значения.
func NewAgentConfig() (*AgentConfig, error) {
	cfg := &AgentConfig{}

	addrFlag := flag.String("a", "localhost:8080", "Metrics server address (host:port)")
	reportSec := flag.Int("r", 10, "Report interval (seconds)")
	pollSec := flag.Int("p", 2, "Poll interval (seconds)")
	keyFlag := flag.String("k", "", "Signing key for HMAC-SHA256 (optional)")
	rateLimit := flag.Int("l", 0, "Maximum number of concurrent outgoing requests (0 = unlimited)")
	cryptoKeyFlag := flag.String("crypto-key", "", "Path to RSA public key PEM file for encryption (optional)")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.ServerAddress = *addrFlag
	cfg.ReportInterval = time.Duration(*reportSec) * time.Second
	cfg.PollInterval = time.Duration(*pollSec) * time.Second
	cfg.Key = *keyFlag
	cfg.RateLimit = *rateLimit
	cfg.CryptoKey = *cryptoKeyFlag

	if addr, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddress = addr
	}
	if val, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid value for REPORT_INTERVAL %q: %w", val, err)
		}
		cfg.ReportInterval = time.Duration(n) * time.Second
	}
	if val, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid value for POLL_INTERVAL %q: %w", val, err)
		}
		cfg.PollInterval = time.Duration(n) * time.Second
	}

	if k, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = k
	}

	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = v
	}

	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Printf("invalid value for RATE_LIMIT %q: %v; keeping previous value %d", v, err, cfg.RateLimit)
		} else {
			cfg.RateLimit = n
		}
	}

	if cfg.RateLimit <= 0 {
		cfg.RateLimit = 1
	}

	if !strings.HasPrefix(cfg.ServerAddress, "http://") && !strings.HasPrefix(cfg.ServerAddress, "https://") {
		cfg.ServerAddress = "http://" + cfg.ServerAddress
	}
	return cfg, nil
}

// ProvideAgentConfig возвращает fx.Option для внедрения AgentConfig.
func ProvideAgentConfig() fx.Option {
	return fx.Provide(NewAgentConfig)
}
