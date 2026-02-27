package agent

import (
	"context"
	"testing"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func newTestAgent(t *testing.T, rateLimit int) *Agent {
	t.Helper()
	logger := zaptest.NewLogger(t)
	sender := newMetricsClient("http://localhost:8080", "", nil)
	return NewAgent(sender, nil, time.Second, time.Second, rateLimit, logger)
}

func TestNewAgent(t *testing.T) {
	a := newTestAgent(t, 5)
	require.NotNil(t, a)
	assert.Equal(t, time.Second, a.pollInterval)
	assert.Equal(t, time.Second, a.reportInterval)
	assert.Equal(t, 5, a.rateLimit)
	assert.NotNil(t, a.collector)
	assert.NotNil(t, a.sender)
	assert.NotNil(t, a.currentMetrics)
}

func TestNewAgent_NegativeRateLimit(t *testing.T) {
	a := newTestAgent(t, -5)
	assert.Equal(t, 0, a.rateLimit)
}

func TestAgent_Collect(t *testing.T) {
	a := newTestAgent(t, 0)

	a.collect()
	a.mx.RLock()
	count := len(a.currentMetrics)
	_, hasPollCount := a.currentMetrics[model.MetricPollCount]
	a.mx.RUnlock()

	assert.True(t, count > 0)
	assert.True(t, hasPollCount)

	a.collect()
	a.mx.RLock()
	pollMetric := a.currentMetrics[model.MetricPollCount]
	a.mx.RUnlock()
	require.NotNil(t, pollMetric.Delta)
	assert.Equal(t, int64(2), *pollMetric.Delta)
}

func TestAgent_SendBatch_Empty(t *testing.T) {
	a := newTestAgent(t, 0)
	a.sendBatch(nil)
	a.sendBatch([]model.Metrics{})
}

func TestAgent_DispatchMetrics_EmptyMap(t *testing.T) {
	a := newTestAgent(t, 0)
	ctx := context.Background()
	a.dispatchMetrics(ctx)
}

func TestAgent_DispatchMetrics_WithData(t *testing.T) {
	a := newTestAgent(t, 0)
	val := 42.0
	a.mx.Lock()
	a.currentMetrics["test"] = model.Metrics{ID: "test", MType: model.Gauge, Value: &val}
	a.mx.Unlock()

	ctx := context.Background()
	a.dispatchMetrics(ctx)
}

func TestAgent_Shutdown_NoMetrics(t *testing.T) {
	a := newTestAgent(t, 0)
	a.Shutdown()
}

func TestAgent_Shutdown_WithMetrics(t *testing.T) {
	a := newTestAgent(t, 0)
	a.collect()
	a.Shutdown()
}

func TestAgent_Shutdown_WithWorkers(t *testing.T) {
	a := newTestAgent(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	a.startWorkers(ctx)
	cancel()
	a.Shutdown()
}

func TestAgent_Run_CancelledContext(t *testing.T) {
	a := newTestAgent(t, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := a.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLoadPublicKey_EmptyPath(t *testing.T) {
	logger := zaptest.NewLogger(t)
	key, err := loadPublicKey("", logger)
	assert.NoError(t, err)
	assert.Nil(t, key)
}

func TestLoadPublicKey_InvalidPath(t *testing.T) {
	logger := zaptest.NewLogger(t)
	key, err := loadPublicKey("/nonexistent/key.pem", logger)
	assert.Error(t, err)
	assert.Nil(t, key)
}
