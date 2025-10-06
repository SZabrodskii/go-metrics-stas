package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/agent"
)

func main() {
	addrFlag := flag.String("a", "localhost:8080", "metrics server address (host:port), e.g. localhost:8080")
	reportSec := flag.Int("r", 10, "report interval in seconds")
	pollSec := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	address := *addrFlag
	if v, ok := os.LookupEnv("ADDRESS"); ok && v != "" {
		address = v
	}
	rSecs := *reportSec
	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Fatalf("invalid REPORT_INTERVAL %q: must be a positive integer seconds", v)
		}
		rSecs = n
	}
	pSecs := *pollSec
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Fatalf("invalid POLL_INTERVAL %q: must be a positive integer seconds", v)
		}
		pSecs = n
	}
	if rSecs <= 0 {
		log.Fatalf("invalid -r: must be > 0 seconds")
	}
	if pSecs <= 0 {
		log.Fatalf("invalid -p: must be > 0 seconds")
	}

	serverURL := ensureHTTPPrefix(address)
	pollInterval := time.Duration(pSecs) * time.Second
	reportInterval := time.Duration(rSecs) * time.Second

	log.Println("Starting metrics agent...")
	log.Printf("Server URL: %s", serverURL)
	log.Printf("Poll interval: %v", pollInterval)
	log.Printf("Report interval: %v", reportInterval)

	metricsAgent := agent.NewAgent(serverURL, pollInterval, reportInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	if err := metricsAgent.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Agent failed: %v", err)
	}

	log.Println("Agent shutdown complete")
}

func ensureHTTPPrefix(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "http://" + addr
}
