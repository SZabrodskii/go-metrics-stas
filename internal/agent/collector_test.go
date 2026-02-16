package agent

import (
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := newMetricsCollector()
	require.NotNil(t, mc)
	assert.Equal(t, int64(0), mc.PollCount)
}

func TestCollectMetrics(t *testing.T) {
	mc := newMetricsCollector()

	metrics := mc.CollectMetrics()
	require.NotEmpty(t, metrics)
	assert.Equal(t, int64(1), mc.PollCount)

	_, hasPollCount := metrics[model.MetricPollCount]
	assert.True(t, hasPollCount)

	_, hasRandomValue := metrics[model.MetricRandomValue]
	assert.True(t, hasRandomValue)

	_, hasAlloc := metrics[model.MetricAlloc]
	assert.True(t, hasAlloc)

	pollMetric := metrics[model.MetricPollCount]
	assert.Equal(t, model.Counter, pollMetric.MType)
	require.NotNil(t, pollMetric.Delta)
	assert.Equal(t, int64(1), *pollMetric.Delta)

	allocMetric := metrics[model.MetricAlloc]
	assert.Equal(t, model.Gauge, allocMetric.MType)
	require.NotNil(t, allocMetric.Value)
}

func TestCollectMetrics_Increments(t *testing.T) {
	mc := newMetricsCollector()

	mc.CollectMetrics()
	metrics := mc.CollectMetrics()

	assert.Equal(t, int64(2), mc.PollCount)
	pollMetric := metrics[model.MetricPollCount]
	require.NotNil(t, pollMetric.Delta)
	assert.Equal(t, int64(2), *pollMetric.Delta)
}

func TestCollectMetrics_AllGaugeMetrics(t *testing.T) {
	mc := newMetricsCollector()
	metrics := mc.CollectMetrics()

	for _, name := range model.GaugeMetrics {
		_, exists := metrics[name]
		assert.True(t, exists, "metric %s should be present", name)
	}
}
