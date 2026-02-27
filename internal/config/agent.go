package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/fx"
)

type agentJSONConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
	GRPCAddress    string `json:"grpc_address"`
}

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
	// GRPCAddress — адрес gRPC сервера (host:port, опционально).
	GRPCAddress string
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
	grpcAddrFlag := flag.String("grpc", "", "gRPC server address (host:port, optional)")

	var configPath string
	flag.StringVar(&configPath, "c", "", "Path to JSON config file")
	flag.StringVar(&configPath, "config", "", "Path to JSON config file (same as -c)")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.ServerAddress = *addrFlag
	cfg.ReportInterval = time.Duration(*reportSec) * time.Second
	cfg.PollInterval = time.Duration(*pollSec) * time.Second
	cfg.Key = *keyFlag
	cfg.RateLimit = *rateLimit
	cfg.CryptoKey = *cryptoKeyFlag
	cfg.GRPCAddress = *grpcAddrFlag

	// Collect explicitly set flags to enforce priority: flags > JSON.
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Config file path: -c/-config flag takes priority over CONFIG env.
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
		}
		var jc agentJSONConfig
		if err := json.Unmarshal(data, &jc); err != nil {
			return nil, fmt.Errorf("parsing config file %s: %w", configPath, err)
		}
		if jc.Address != "" && !setFlags["a"] {
			cfg.ServerAddress = jc.Address
		}
		if jc.ReportInterval != "" && !setFlags["r"] {
			d, err := time.ParseDuration(jc.ReportInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid report_interval in config file: %w", err)
			}
			cfg.ReportInterval = d
		}
		if jc.PollInterval != "" && !setFlags["p"] {
			d, err := time.ParseDuration(jc.PollInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid poll_interval in config file: %w", err)
			}
			cfg.PollInterval = d
		}
		if jc.CryptoKey != "" && !setFlags["crypto-key"] {
			cfg.CryptoKey = jc.CryptoKey
		}
		if jc.GRPCAddress != "" && !setFlags["grpc"] {
			cfg.GRPCAddress = jc.GRPCAddress
		}
	}

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
	if v, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		cfg.GRPCAddress = v
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
