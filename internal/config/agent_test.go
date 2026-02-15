package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearAgentEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL", "KEY", "CRYPTO_KEY", "RATE_LIMIT", "CONFIG"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestAgentConfig_Defaults(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8080", cfg.ServerAddress)
	assert.Equal(t, 10*time.Second, cfg.ReportInterval)
	assert.Equal(t, 2*time.Second, cfg.PollInterval)
	assert.Empty(t, cfg.Key)
	assert.Equal(t, 1, cfg.RateLimit)
	assert.Empty(t, cfg.CryptoKey)
}

func TestAgentConfig_JSONConfig(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{
		"address": "10.0.0.1:5000",
		"report_interval": "30s",
		"poll_interval": "5s",
		"crypto_key": "/tmp/pub.pem"
	}`)

	os.Args = []string{"test", "-c", path}

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://10.0.0.1:5000", cfg.ServerAddress)
	assert.Equal(t, 30*time.Second, cfg.ReportInterval)
	assert.Equal(t, 5*time.Second, cfg.PollInterval)
	assert.Equal(t, "/tmp/pub.pem", cfg.CryptoKey)
}

func TestAgentConfig_JSONViaConfigEnv(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{"address": "192.168.1.1:7000"}`)
	t.Setenv("CONFIG", path)

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://192.168.1.1:7000", cfg.ServerAddress)
}

func TestAgentConfig_FlagOverridesJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{
		"address": "json-host:1111",
		"report_interval": "30s"
	}`)
	os.Args = []string{"test", "-c", path, "-a", "flag-host:2222", "-r", "5"}

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://flag-host:2222", cfg.ServerAddress)
	assert.Equal(t, 5*time.Second, cfg.ReportInterval)
}

func TestAgentConfig_EnvOverridesJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{
		"address": "json-host:1111",
		"poll_interval": "30s"
	}`)
	os.Args = []string{"test", "-c", path}
	t.Setenv("ADDRESS", "env-host:3333")
	t.Setenv("POLL_INTERVAL", "15")

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://env-host:3333", cfg.ServerAddress)
	assert.Equal(t, 15*time.Second, cfg.PollInterval)
}

func TestAgentConfig_InvalidJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{broken`)
	os.Args = []string{"test", "-c", path}

	_, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestAgentConfig_MissingConfigFile(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	os.Args = []string{"test", "-c", "/nonexistent/agent.json"}

	_, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestAgentConfig_InvalidReportIntervalInJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{"report_interval": "bad"}`)
	os.Args = []string{"test", "-c", path}

	_, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid report_interval")
}

func TestAgentConfig_InvalidPollIntervalInJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{"poll_interval": "bad"}`)
	os.Args = []string{"test", "-c", path}

	_, err := NewAgentConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid poll_interval")
}

func TestAgentConfig_NoConfigFile(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8080", cfg.ServerAddress)
	assert.Equal(t, 10*time.Second, cfg.ReportInterval)
	assert.Equal(t, 2*time.Second, cfg.PollInterval)
}

func TestAgentConfig_FullPriority(t *testing.T) {
	resetFlagsAndArgs()
	clearAgentEnv(t)

	path := writeTempJSON(t, `{
		"address": "json:1000",
		"report_interval": "30s",
		"poll_interval": "15s",
		"crypto_key": "/json/key.pem"
	}`)

	os.Args = []string{"test", "-c", path, "-r", "60"}
	t.Setenv("POLL_INTERVAL", "7")

	cfg, err := NewAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://json:1000", cfg.ServerAddress)
	assert.Equal(t, 60*time.Second, cfg.ReportInterval)
	assert.Equal(t, 7*time.Second, cfg.PollInterval)
	assert.Equal(t, "/json/key.pem", cfg.CryptoKey)
}
