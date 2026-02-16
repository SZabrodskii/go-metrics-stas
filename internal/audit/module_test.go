package audit

import (
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewPublisherWithConfig_Disabled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{}

	pub := NewPublisherWithConfig(cfg, logger)
	assert.Nil(t, pub)
}

func TestNewPublisherWithConfig_FileOnly(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{
		AuditFile: "/tmp/test-audit.log",
	}

	pub := NewPublisherWithConfig(cfg, logger)
	require.NotNil(t, pub)
	assert.Len(t, pub.observers, 1)
	assert.Contains(t, pub.observers, "file_auditor_/tmp/test-audit.log")
}

func TestNewPublisherWithConfig_HTTPOnly(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{
		AuditURL: "http://localhost:9999/audit",
	}

	pub := NewPublisherWithConfig(cfg, logger)
	require.NotNil(t, pub)
	assert.Len(t, pub.observers, 1)
	assert.Contains(t, pub.observers, "http_auditor_http://localhost:9999/audit")
}

func TestNewPublisherWithConfig_Both(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := &config.ServerConfig{
		AuditFile: "/tmp/audit.log",
		AuditURL:  "http://localhost:9999/audit",
	}

	pub := NewPublisherWithConfig(cfg, logger)
	require.NotNil(t, pub)
	assert.Len(t, pub.observers, 2)
}
