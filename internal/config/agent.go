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

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddress  string
	Key            string
	RateLimit      int
}

func NewAgentConfig() *AgentConfig {
	cfg := &AgentConfig{}

	addrFlag := flag.String("a", "localhost:8080", "Metrics server address (host:port)")
	reportSec := flag.Int("r", 10, "Report interval (seconds)")
	pollSec := flag.Int("p", 2, "Poll interval (seconds)")
	keyFlag := flag.String("k", "", "Signing key for HMAC-SHA256 (optional)")
	rateLimit := flag.Int("l", 1, "Maximum number of concurrent outgoing requests")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.ServerAddress = *addrFlag
	cfg.ReportInterval = time.Duration(*reportSec) * time.Second
	cfg.PollInterval = time.Duration(*pollSec) * time.Second
	cfg.Key = *keyFlag
	cfg.RateLimit = *rateLimit

	if addr, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddress = addr
	}
	if val, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		n, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("invalid value for REPORT_INTERVAL %q: %v", val, err)
		}
		cfg.ReportInterval = time.Duration(n) * time.Second
	}
	if val, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		n, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("invalid value for POLL_INTERVAL %q: %v", val, err)
		}
		cfg.PollInterval = time.Duration(n) * time.Second
	}

	if k, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = k
	}

	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Fatalf("invalid value for RATE_LIMIT %q: %v", v, err)
		}
		cfg.RateLimit = n
	}

	if cfg.RateLimit <= 0 {
		cfg.RateLimit = 1
	}

	if !strings.HasPrefix(cfg.ServerAddress, "http://") && !strings.HasPrefix(cfg.ServerAddress, "https://") {
		cfg.ServerAddress = "http://" + cfg.ServerAddress
	}
	return cfg
}

func ProvideAgentConfig() fx.Option {
	return fx.Provide(NewAgentConfig)
}
