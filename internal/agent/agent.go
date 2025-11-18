package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Agent struct {
	collector      *metricsCollector
	client         *metricsClient
	pollInterval   time.Duration
	reportInterval time.Duration
	currentMetrics map[string]model.Metrics
	mx             sync.RWMutex
	logger         *zap.Logger

	rateLimit int
	jobs      chan model.Metrics
	workersWG sync.WaitGroup
}

func NewAgent(serverURL string, pollInterval time.Duration, reportInterval time.Duration, key string, rateLimit int, logger *zap.Logger) *Agent {
	if rateLimit <= 0 {
		rateLimit = 1
	}

	return &Agent{
		collector:      newMetricsCollector(),
		client:         newMetricsClient(serverURL, key),
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		currentMetrics: make(map[string]model.Metrics),
		rateLimit:      rateLimit,
		logger:         logger,
	}
}

func (a *Agent) startWorkers(ctx context.Context) {
	a.logger.Info("starting workers pool", zap.Int("rateLimit", a.rateLimit))

	a.jobs = make(chan model.Metrics, a.rateLimit)

	for i := 0; i < a.rateLimit; i++ {
		a.workersWG.Add(1)

		go func(id int) {
			defer a.workersWG.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case m, ok := <-a.jobs:
					if !ok {
						return
					}
					if err := a.client.SendMetric(m); err != nil {
						a.logger.Error("failed to send metric",
							zap.String("id", m.ID),
							zap.Error(err),
							zap.Int("worker", id))
					}
				}
			}
		}(i + 1)
	}
}

func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("Starting metrics agent",
		zap.Duration("pollInterval", a.pollInterval),
		zap.Duration("reportInterval", a.reportInterval),
		zap.Int("rateLimit", a.rateLimit),
	)

	pollTicker := time.NewTicker(a.pollInterval)
	defer pollTicker.Stop()

	a.startWorkers(ctx)

	go a.dispatchLoop(ctx)

	go a.collectSystemLoop(ctx)

	a.collect()

	for {
		select {
		case <-ctx.Done():
			close(a.jobs)
			a.workersWG.Wait()
			return ctx.Err()
		case <-pollTicker.C:
			a.collect()
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
	if len(a.currentMetrics) == 0 {
		a.mx.RUnlock()
		a.logger.Info("No metrics to send")
		return
	}

	batch := make([]model.Metrics, 0, len(a.currentMetrics))
	for _, m := range a.currentMetrics {
		batch = append(batch, m)
	}
	a.mx.RUnlock()

	if err := a.client.SendBatch(batch); err != nil {
		a.logger.Warn("Batch send failed, fallback to single", zap.Error(err))
		for _, m := range batch {
			if err := a.client.SendMetric(m); err != nil {
				a.logger.Error("Failed to send metric", zap.String("id", m.ID), zap.Error(err))
			}
		}
	} else {
		a.logger.Info("Successfully sent metrics to server", zap.Int("count", len(batch)))
	}

}

func (a *Agent) dispatchLoop(ctx context.Context) {
	reportTicker := time.NewTicker(a.reportInterval)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-reportTicker.C:
			a.dispatchMetrics(ctx)
		}
	}
}

func (a *Agent) dispatchMetrics(ctx context.Context) {
	a.logger.Info("Dispatching metrics to workers...")

	a.mx.RLock()
	if len(a.currentMetrics) == 0 {
		a.mx.RUnlock()
		a.logger.Info("No metrics to send")
		return
	}

	snapshot := make([]model.Metrics, 0, len(a.currentMetrics))

	for _, m := range a.currentMetrics {
		snapshot = append(snapshot, m)
	}
	a.mx.RUnlock()

	for _, m := range snapshot {
		select {
		case <-ctx.Done():
			return
		case a.jobs <- m:
		}
	}
}

func (a *Agent) collectSystemLoop(ctx context.Context) {
	ticker := time.NewTicker(a.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.collectSystemMetrics()
		}
	}
}

func (a *Agent) collectSystemMetrics() {
	virtualMemoryStat, err := mem.VirtualMemory()
	if err != nil {
		a.logger.Warn("Failed to get memory metrics", zap.Error(err))
	} else {
		total := float64(virtualMemoryStat.Total)
		free := float64(virtualMemoryStat.Free)

		a.mx.Lock()
		a.currentMetrics["TotalMemory"] = model.Metrics{
			ID:    "TotalMemory",
			MType: model.Gauge,
			Value: &total,
		}
		a.currentMetrics["FreeMemory"] = model.Metrics{
			ID:    "FreeMemory",
			MType: model.Gauge,
			Value: &free,
		}
		a.mx.Unlock()
	}

	percentages, err := cpu.Percent(0, true)
	if err != nil {
		a.logger.Warn("Failed to get cpu metrics", zap.Error(err))
		return
	}

	a.mx.Lock()
	defer a.mx.Unlock()

	for i, p := range percentages {
		v := p
		id := fmt.Sprintf("CPUutilization%d", i+1)
		a.currentMetrics[id] = model.Metrics{
			ID:    id,
			MType: model.Gauge,
			Value: &v,
		}
	}
}

var Module = fx.Options(
	fx.Provide(
		func(cfg *config.AgentConfig, logger *zap.Logger) *Agent {
			return NewAgent(cfg.ServerAddress, cfg.PollInterval, cfg.ReportInterval, cfg.Key, cfg.RateLimit, logger)
		},
	),

	fx.Invoke(runAgent),
)

func runAgent(lc fx.Lifecycle, agent *Agent, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := agent.Run(ctx); err != nil && err != context.Canceled {
					logger.Fatal("Failed to start agent", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("Agent stopped")
			cancel()
			return nil
		},
	})
}
