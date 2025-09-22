package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
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

	if *reportSec <= 0 {
		log.Fatalf("invalid -r: must be > 0 seconds")
	}
	if *pollSec <= 0 {
		log.Fatalf("invalid -p: must be > 0 seconds")
	}

	serverURL := ensureHTTPPrefix(*addrFlag)
	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

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
