package agent

import (
	"context"
	"log"
	"time"
	"sync"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

type Agent struct {
	collector *MetricsCollector
	client    *MetricsClient
	pollInterval  time.Duration
	reportInterval time.Duration
	currentMetrics map[string]model.Metrics
	mx sync.RWMutex
}

func NewAgent(serverURL string, pollInterval time.Duration, reportInterval time.Duration) *Agent {
	return &Agent{
		collector:    NewMetricsCollector(),
		client:      NewMetricsClient(serverURL),
		pollInterval: pollInterval,
		reportInterval: reportInterval,
		currentMetrics: make(map[string]model.Metrics),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	log.Printf("Starting metrics agent, sending metrics every %v", a.pollInterval)

	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	a.collect()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Agent stopped")
			return ctx.Err()
		case <-pollTicker.C:
			a.collect()
		case <-reportTicker.C:
			a.send()
		}
	}
}

func (a *Agent) collect() {
	log.Println("Collecting metrics...")
	
	a.mx.Lock()
	defer a.mx.Unlock()

	metrics := a.collector.CollectMetrics()
	for k, v := range metrics {
		a.currentMetrics[k] = v
	}
	
	log.Printf("Collected %d metrics", len(metrics))
}
func (a *Agent) send() {
	log.Println("Sending metrics to server...")
	
	a.mx.RLock()
	defer a.mx.RUnlock()

	if err := a.client.SendMetrics(a.currentMetrics); err != nil {
		log.Printf("Error sending metrics: %v", err)
	} else {
		log.Printf("Successfully sent %d metrics", len(a.currentMetrics))
	}
}

