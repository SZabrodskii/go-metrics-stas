package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestFileAuditor_GetID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	fa := NewFileAuditor("/tmp/audit.log", logger)
	assert.Equal(t, "file_auditor_/tmp/audit.log", fa.GetID())
}

func TestFileAuditor_Update(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := filepath.Join(t.TempDir(), "audit.log")
	fa := NewFileAuditor(path, logger)

	event := AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"Alloc", "TotalAlloc"},
		IPAddress: "192.168.1.1",
	}

	err := fa.Update(event)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var written AuditEvent
	err = json.Unmarshal([]byte(strings.TrimSpace(string(data))), &written)
	require.NoError(t, err)

	assert.Equal(t, event.Timestamp, written.Timestamp)
	assert.Equal(t, event.Metrics, written.Metrics)
	assert.Equal(t, event.IPAddress, written.IPAddress)
}

func TestFileAuditor_UpdateAppends(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := filepath.Join(t.TempDir(), "audit.log")
	fa := NewFileAuditor(path, logger)

	for i := 0; i < 3; i++ {
		err := fa.Update(AuditEvent{
			Timestamp: time.Now().Unix(),
			Metrics:   []string{"metric"},
			IPAddress: "10.0.0.1",
		})
		require.NoError(t, err)
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 3)
}

func TestFileAuditor_InvalidPath(t *testing.T) {
	logger := zaptest.NewLogger(t)
	fa := NewFileAuditor("/nonexistent/dir/audit.log", logger)

	err := fa.Update(AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"m"},
		IPAddress: "1.2.3.4",
	})
	assert.Error(t, err)
}

func TestFileAuditor_ConcurrentWrites(t *testing.T) {
	logger := zaptest.NewLogger(t)
	path := filepath.Join(t.TempDir(), "audit.log")
	fa := NewFileAuditor(path, logger)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := fa.Update(AuditEvent{
				Timestamp: time.Now().Unix(),
				Metrics:   []string{"concurrent"},
				IPAddress: "10.0.0.1",
			})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Len(t, lines, 10)
}
