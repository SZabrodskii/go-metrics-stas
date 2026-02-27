// Package agent реализует агент для сбора и отправки метрик на сервер.
package agent

import (
	"context"
	"crypto/rsa"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	appcrypto "github.com/SZabrodskii/go-metrics-stas/internal/crypto"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrCreateGRPCClient = errors.New("create gRPC client")
	ErrLoadPublicKey    = errors.New("load RSA public key")
)

type BatchSender interface {
	SendBatch(metrics []model.Metrics) error
}

// Agent собирает метрики из runtime и системы, отправляя их на сервер.
// Поддерживает параллельную отправку через пул воркеров (rate limiting).
type Agent struct {
	collector      *metricsCollector
	sender         BatchSender
	grpcClient     *grpcMetricsClient
	pollInterval   time.Duration
	reportInterval time.Duration
	currentMetrics map[string]model.Metrics
	mx             sync.RWMutex
	logger         *zap.Logger

	rateLimit int
	jobs      chan []model.Metrics
	workersWG sync.WaitGroup
}

func NewAgent(sender BatchSender, grpcClient *grpcMetricsClient, pollInterval time.Duration, reportInterval time.Duration, rateLimit int, logger *zap.Logger) *Agent {
	if rateLimit < 0 {
		rateLimit = 0
	}

	return &Agent{
		collector:      newMetricsCollector(),
		sender:         sender,
		grpcClient:     grpcClient,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		currentMetrics: make(map[string]model.Metrics),
		rateLimit:      rateLimit,
		logger:         logger,
	}
}

func (a *Agent) startWorkers(ctx context.Context) {
	if a.rateLimit <= 0 {
		a.logger.Info("rate limiting disabled; sending batches synchronously")
		return
	}
	a.logger.Info("starting workers pool", zap.Int("rateLimit", a.rateLimit))

	a.jobs = make(chan []model.Metrics, a.rateLimit)

	for i := 0; i < a.rateLimit; i++ {
		a.workersWG.Add(1)

		go func(id int) {
			defer a.workersWG.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case batch, ok := <-a.jobs:
					if !ok {
						return
					}
					if len(batch) == 0 {
						continue
					}
					if err := a.sender.SendBatch(batch); err != nil {
						a.logger.Error("failed to send batch",
							zap.Int("batch_size:", len(batch)),
							zap.Int("worker", id),
							zap.Error(err))
					} else {
						a.logger.Info("batch sent successfully",
							zap.Int("batch_size:", len(batch)),
							zap.Int("worker", id),
						)
					}
				}
			}
		}(i + 1)
	}
}

// Run запускает главный цикл агента.
// Собирает метрики с интервалом pollInterval и отправляет с интервалом reportInterval.
// Блокируется до отмены контекста.
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
			return ctx.Err()
		case <-pollTicker.C:
			a.collect()
		}
	}
}

// Shutdown выполняет graceful shutdown агента
func (a *Agent) Shutdown() {
	a.logger.Info("Sending final metrics before shutdown")

	// Отправляем накопленные метрики синхронно
	a.mx.RLock()
	if len(a.currentMetrics) > 0 {
		snapshot := make([]model.Metrics, 0, len(a.currentMetrics))
		for _, m := range a.currentMetrics {
			snapshot = append(snapshot, m)
		}
		a.mx.RUnlock()

		a.sendBatch(snapshot)
	} else {
		a.mx.RUnlock()
	}

	// Закрываем канал воркеров и ждём их завершения
	if a.rateLimit > 0 && a.jobs != nil {
		close(a.jobs)
		a.workersWG.Wait()
	}

	if a.grpcClient != nil {
		_ = a.grpcClient.Close()
	}

	a.logger.Info("Final metrics sent, workers stopped")
}

func (a *Agent) sendBatch(batch []model.Metrics) {
	if len(batch) == 0 {
		a.logger.Info("batch size is zero; no metrics to send")
		return
	}

	if err := a.sender.SendBatch(batch); err != nil {
		a.logger.Error("failed to send batch",
			zap.Int("count:", len(batch)),
			zap.Error(err))
	} else {
		a.logger.Info("batch sent successfully to server",
			zap.Int("count:", len(batch)))
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

	if a.rateLimit <= 0 || a.jobs == nil {
		a.sendBatch(snapshot)
		return
	}

	select {
	case <-ctx.Done():
		return
	case a.jobs <- snapshot:
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
		id := "CPUutilization" + strconv.Itoa(i+1)
		a.currentMetrics[id] = model.Metrics{
			ID:    id,
			MType: model.Gauge,
			Value: &v,
		}
	}
}

// Module предоставляет fx модуль для внедрения зависимостей агента.
var Module = fx.Options(
	fx.Provide(
		func(cfg *config.AgentConfig, logger *zap.Logger) (*Agent, error) {
			var sender BatchSender
			var grpcCl *grpcMetricsClient

			if cfg.GRPCAddress != "" {
				localIP := resolveLocalIP()
				var err error
				grpcCl, err = newGRPCMetricsClient(cfg.GRPCAddress, localIP)
				if err != nil {
					return nil, errors.Join(ErrCreateGRPCClient, err)
				}
				sender = grpcCl
				logger.Info("Agent will send metrics via gRPC", zap.String("address", cfg.GRPCAddress))
			} else {
				pubKey, err := loadPublicKey(cfg.CryptoKey, logger)
				if err != nil {
					return nil, err
				}
				sender = newMetricsClient(cfg.ServerAddress, cfg.Key, pubKey)
				logger.Info("Agent will send metrics via HTTP", zap.String("address", cfg.ServerAddress))
			}

			return NewAgent(sender, grpcCl, cfg.PollInterval, cfg.ReportInterval, cfg.RateLimit, logger), nil
		},
	),

	fx.Invoke(runAgent),
)

func loadPublicKey(path string, logger *zap.Logger) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, nil
	}
	key, err := appcrypto.LoadPublicKey(path)
	if err != nil {
		logger.Error("failed to load RSA public key", zap.String("path", path), zap.Error(err))
		return nil, errors.Join(ErrLoadPublicKey, err)
	}
	logger.Info("RSA public key loaded for encryption", zap.String("path", path))
	return key, nil
}

func runAgent(lc fx.Lifecycle, agent *Agent, logger *zap.Logger, shutdowner fx.Shutdowner) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := agent.Run(ctx); err != nil && err != context.Canceled {
					logger.Error("Failed to start agent", zap.Error(err))
					_ = shutdowner.Shutdown()
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Initiating agent graceful shutdown")
			cancel()

			shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
			defer shutdownCancel()

			done := make(chan struct{})
			go func() {
				agent.Shutdown()
				close(done)
			}()

			select {
			case <-done:
				logger.Info("Agent graceful shutdown completed")
			case <-shutdownCtx.Done():
				logger.Warn("Agent shutdown timed out")
			}
			return nil
		},
	})
}
