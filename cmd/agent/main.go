package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/SZabrodskii/go-metrics-stas/internal/agent"
)

func main() {
	serverURL := "http://localhost:8080"
	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second 

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
