package agent

import (
	"context"
	"sync"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"go.uber.org/zap"
)

type Agent struct {
	collector      *MetricsCollector
	client         *MetricsClient
	pollInterval   time.Duration
	reportInterval time.Duration
	currentMetrics map[string]model.Metrics
	mx             sync.RWMutex
	logger         *zap.Logger
}

func NewAgent(serverURL string, pollInterval time.Duration, reportInterval time.Duration, logger *zap.Logger) *Agent {
	return &Agent{
		collector:      NewMetricsCollector(),
		client:         NewMetricsClient(serverURL),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		currentMetrics: make(map[string]model.Metrics),
		logger:         logger,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("Starting metrics agent",
		zap.Duration("pollInterval", a.pollInterval),
		zap.Duration("reportInterval", a.reportInterval),
	)

	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	a.collect()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pollTicker.C:
			a.collect()
		case <-reportTicker.C:
			a.send()
		}
	}
}

func (a *Agent) collect() {
	a.logger.Debug("Collecting metrics...")

	a.mx.Lock()
	defer a.mx.Unlock()

	metrics := a.collector.CollectMetrics()
	for k, v := range metrics {
		a.currentMetrics[k] = v
	}

	a.logger.Info("Metrics collected", zap.Int("count", len(metrics)))
}
func (a *Agent) send() {
	a.logger.Info("Sending metrics to server...")

	a.mx.RLock()
	defer a.mx.RUnlock()

	if err := a.client.SendMetrics(a.currentMetrics); err != nil {
		a.logger.Error("Error sending metrics", zap.Error(err))
	} else {
		a.logger.Info("Successfully sent metrics", zap.Int("count", len(a.currentMetrics)))
	}
}
