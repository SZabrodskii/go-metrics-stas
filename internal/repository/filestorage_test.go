package repository

import (
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestFileBackedStorage_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "metrics.json")
	logger := zaptest.NewLogger(t)

	s := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		syncWrite:  false,
		stopCh:     make(chan struct{}),
	}

	s.MemStorage.UpdateGauge("cpu", 75.5)
	s.MemStorage.UpdateCounter("hits", 10)

	err := s.saveToFile()
	require.NoError(t, err)

	s2 := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		syncWrite:  false,
		stopCh:     make(chan struct{}),
	}

	err = s2.loadFromFile()
	require.NoError(t, err)

	g, err := s2.GetGauge("cpu")
	require.NoError(t, err)
	assert.InDelta(t, 75.5, g, 0.001)

	c, err := s2.GetCounter("hits")
	require.NoError(t, err)
	assert.Equal(t, int64(10), c)
}

func TestFileBackedStorage_LoadFromFile_NotExist(t *testing.T) {
	logger := zaptest.NewLogger(t)
	s := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   "/nonexistent/path/metrics.json",
		logger:     logger,
		stopCh:     make(chan struct{}),
	}

	err := s.loadFromFile()
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestFileBackedStorage_SyncWrite_Gauge(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "metrics.json")
	logger := zaptest.NewLogger(t)

	s := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		syncWrite:  true,
		stopCh:     make(chan struct{}),
	}

	s.UpdateGauge("temp", 36.6)

	_, err := os.Stat(fp)
	assert.NoError(t, err)

	s2 := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
	require.NoError(t, s2.loadFromFile())
	g, err := s2.GetGauge("temp")
	require.NoError(t, err)
	assert.InDelta(t, 36.6, g, 0.001)
}

func TestFileBackedStorage_SyncWrite_Counter(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "metrics.json")
	logger := zaptest.NewLogger(t)

	s := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		syncWrite:  true,
		stopCh:     make(chan struct{}),
	}

	s.UpdateCounter("req", 5)

	s2 := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
	require.NoError(t, s2.loadFromFile())
	c, err := s2.GetCounter("req")
	require.NoError(t, err)
	assert.Equal(t, int64(5), c)
}

func TestFileBackedStorage_UpdateBatch(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "metrics.json")
	logger := zaptest.NewLogger(t)

	s := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   fp,
		logger:     logger,
		syncWrite:  true,
		stopCh:     make(chan struct{}),
	}

	val := 42.0
	delta := int64(7)
	batch := []model.Metrics{
		{ID: "g1", MType: model.Gauge, Value: &val},
		{ID: "c1", MType: model.Counter, Delta: &delta},
	}
	err := s.UpdateBatch(batch)
	require.NoError(t, err)

	g, err := s.GetGauge("g1")
	require.NoError(t, err)
	assert.InDelta(t, 42.0, g, 0.001)

	c, err := s.GetCounter("c1")
	require.NoError(t, err)
	assert.Equal(t, int64(7), c)
}

func TestFileBackedStorage_UpdateBatch_Empty(t *testing.T) {
	logger := zaptest.NewLogger(t)
	s := &fileBackedStorage{
		MemStorage: NewMemStorage(),
		filePath:   "/tmp/test.json",
		logger:     logger,
		stopCh:     make(chan struct{}),
	}

	err := s.UpdateBatch(nil)
	assert.NoError(t, err)

	err = s.UpdateBatch([]model.Metrics{})
	assert.NoError(t, err)
}

func TestIsPGConnException(t *testing.T) {
	assert.False(t, isPGConnException(nil))
	assert.False(t, isPGConnException(errors.New("generic error")))
	assert.True(t, isPGConnException(driver.ErrBadConn))

	pgErr := &pgconn.PgError{Code: "08006"}
	assert.True(t, isPGConnException(pgErr))

	pgErr2 := &pgconn.PgError{Code: "40001"}
	assert.True(t, isPGConnException(pgErr2))

	pgErr3 := &pgconn.PgError{Code: "23505"}
	assert.False(t, isPGConnException(pgErr3))
}

func TestRetryPG_Success(t *testing.T) {
	calls := 0
	err := retryPG(func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryPG_NonRetryableError(t *testing.T) {
	calls := 0
	err := retryPG(func() error {
		calls++
		return errors.New("not retryable")
	})
	assert.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestNewMemStorage(t *testing.T) {
	ms := NewMemStorage()
	require.NotNil(t, ms)
	require.NotNil(t, ms.gauges)
	require.NotNil(t, ms.counters)
}

func TestMemStorage_GetGauge_NotFound(t *testing.T) {
	ms := NewMemStorage()
	_, err := ms.GetGauge("missing")
	assert.ErrorIs(t, err, ErrGaugeNotFound)
}

func TestMemStorage_GetCounter_NotFound(t *testing.T) {
	ms := NewMemStorage()
	_, err := ms.GetCounter("missing")
	assert.ErrorIs(t, err, ErrCounterNotFound)
}

func TestMemStorage_UpdateBatch_Empty(t *testing.T) {
	ms := NewMemStorage()
	err := ms.UpdateBatch(nil)
	assert.NoError(t, err)
}
