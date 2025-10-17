package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type AgentConfig struct {
	PollInterval   time.Duration
	ReportInterval time.Duration
	ServerAddress  string
}

func NewAgentConfig() *AgentConfig {
	cfg := &AgentConfig{}

	addrFlag := flag.String("a", "localhost:8080", "Metrics server address (host:port)")
	reportSec := flag.Int("r", 10, "Report interval (seconds)")
	pollSec := flag.Int("p", 2, "Poll interval (seconds)")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.ServerAddress = *addrFlag
	cfg.ReportInterval = time.Duration(*reportSec) * time.Second
	cfg.PollInterval = time.Duration(*pollSec) * time.Second

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
	if !strings.HasPrefix(cfg.ServerAddress, "http://") && !strings.HasPrefix(cfg.ServerAddress, "https://") {
		cfg.ServerAddress = "http://" + cfg.ServerAddress
	}
	return cfg
}
