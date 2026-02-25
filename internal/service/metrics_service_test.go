package service

import (
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/SZabrodskii/go-metrics-stas/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService() MetricsService {
	return NewMetricsService(repository.NewMemStorage(), zap.NewNop())
}

func TestUpdateGauge(t *testing.T) {
	svc := newTestService()

	t.Run("success", func(t *testing.T) {
		err := svc.UpdateGauge("temp", 36.6)
		require.NoError(t, err)

		val, err := svc.GetGauge("temp")
		require.NoError(t, err)
		assert.InDelta(t, 36.6, val, 0.001)
	})

	t.Run("empty id", func(t *testing.T) {
		err := svc.UpdateGauge("", 1.0)
		assert.ErrorIs(t, err, ErrInvalidMetricID)
	})
}

func TestUpdateCounter(t *testing.T) {
	svc := newTestService()

	t.Run("success", func(t *testing.T) {
		newVal, err := svc.UpdateCounter("hits", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(5), newVal)
	})

	t.Run("accumulation", func(t *testing.T) {
		newVal, err := svc.UpdateCounter("hits", 3)
		require.NoError(t, err)
		assert.Equal(t, int64(8), newVal)
	})

	t.Run("empty id", func(t *testing.T) {
		_, err := svc.UpdateCounter("", 1)
		assert.ErrorIs(t, err, ErrInvalidMetricID)
	})
}

func TestGetGauge(t *testing.T) {
	svc := newTestService()

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetGauge("missing")
		assert.ErrorIs(t, err, ErrMetricNotFound)
	})

	t.Run("success", func(t *testing.T) {
		_ = svc.UpdateGauge("cpu", 75.5)
		val, err := svc.GetGauge("cpu")
		require.NoError(t, err)
		assert.InDelta(t, 75.5, val, 0.001)
	})
}

func TestGetCounter(t *testing.T) {
	svc := newTestService()

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetCounter("missing")
		assert.ErrorIs(t, err, ErrMetricNotFound)
	})

	t.Run("success", func(t *testing.T) {
		_, _ = svc.UpdateCounter("reqs", 10)
		val, err := svc.GetCounter("reqs")
		require.NoError(t, err)
		assert.Equal(t, int64(10), val)
	})
}

func TestGetAllMetrics(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		svc := newTestService()
		m, err := svc.GetAllMetrics()
		require.NoError(t, err)
		assert.Empty(t, m)
	})

	t.Run("with data", func(t *testing.T) {
		svc := newTestService()
		_ = svc.UpdateGauge("g1", 1.0)
		_, _ = svc.UpdateCounter("c1", 5)

		m, err := svc.GetAllMetrics()
		require.NoError(t, err)
		assert.Len(t, m, 2)
		assert.Contains(t, m, "g1")
		assert.Contains(t, m, "c1")
	})
}

func TestUpdateBatch(t *testing.T) {
	t.Run("valid batch", func(t *testing.T) {
		svc := newTestService()
		val := 1.0
		delta := int64(5)
		batch := []model.Metrics{
			{ID: "g1", MType: "gauge", Value: &val},
			{ID: "c1", MType: "counter", Delta: &delta},
		}
		err := svc.UpdateBatch(batch)
		require.NoError(t, err)

		g, err := svc.GetGauge("g1")
		require.NoError(t, err)
		assert.InDelta(t, 1.0, g, 0.001)

		c, err := svc.GetCounter("c1")
		require.NoError(t, err)
		assert.Equal(t, int64(5), c)
	})

	t.Run("empty id", func(t *testing.T) {
		svc := newTestService()
		val := 1.0
		batch := []model.Metrics{{MType: "gauge", Value: &val}}
		err := svc.UpdateBatch(batch)
		assert.ErrorIs(t, err, ErrInvalidMetricID)
	})

	t.Run("gauge without value", func(t *testing.T) {
		svc := newTestService()
		batch := []model.Metrics{{ID: "x", MType: "gauge"}}
		err := svc.UpdateBatch(batch)
		assert.ErrorIs(t, err, ErrMissingValue)
	})

	t.Run("counter without delta", func(t *testing.T) {
		svc := newTestService()
		batch := []model.Metrics{{ID: "x", MType: "counter"}}
		err := svc.UpdateBatch(batch)
		assert.ErrorIs(t, err, ErrMissingDelta)
	})

	t.Run("invalid type", func(t *testing.T) {
		svc := newTestService()
		val := 1.0
		batch := []model.Metrics{{ID: "x", MType: "bad", Value: &val}}
		err := svc.UpdateBatch(batch)
		assert.ErrorIs(t, err, ErrInvalidMetricType)
	})
}
