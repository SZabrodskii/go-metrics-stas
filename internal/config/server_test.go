package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlagsAndArgs() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"test"}
}

func clearServerEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH", "RESTORE", "DATABASE_DSN", "KEY", "CRYPTO_KEY", "AUDIT_FILE", "AUDIT_URL", "CONFIG"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.json")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestServerConfig_Defaults(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.ListenAddress)
	assert.Equal(t, 300*time.Second, cfg.StoreInterval)
	assert.True(t, cfg.Restore)
	assert.Empty(t, cfg.FileStoragePath)
	assert.Empty(t, cfg.DatabaseDSN)
	assert.Empty(t, cfg.CryptoKey)
}

func TestServerConfig_JSONConfig(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{
		"address": "0.0.0.0:9090",
		"restore": false,
		"store_interval": "10s",
		"store_file": "/tmp/metrics.db",
		"database_dsn": "postgres://localhost/test",
		"crypto_key": "/tmp/key.pem"
	}`)

	os.Args = []string{"test", "-c", path}

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0:9090", cfg.ListenAddress)
	assert.False(t, cfg.Restore)
	assert.Equal(t, 10*time.Second, cfg.StoreInterval)
	assert.Equal(t, "/tmp/metrics.db", cfg.FileStoragePath)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseDSN)
	assert.Equal(t, "/tmp/key.pem", cfg.CryptoKey)
}

func TestServerConfig_JSONViaConfigEnv(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{"address": "10.0.0.1:3000"}`)
	t.Setenv("CONFIG", path)

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "10.0.0.1:3000", cfg.ListenAddress)
}

func TestServerConfig_FlagOverridesJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{"address": "json-host:1111"}`)
	os.Args = []string{"test", "-c", path, "-a", "flag-host:2222"}

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "flag-host:2222", cfg.ListenAddress)
}

func TestServerConfig_EnvOverridesJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{"address": "json-host:1111"}`)
	os.Args = []string{"test", "-c", path}
	t.Setenv("ADDRESS", "env-host:3333")

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "env-host:3333", cfg.ListenAddress)
}

func TestServerConfig_InvalidJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{invalid`)
	os.Args = []string{"test", "-c", path}

	_, err := NewServerConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestServerConfig_MissingConfigFile(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	os.Args = []string{"test", "-c", "/nonexistent/config.json"}

	_, err := NewServerConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading config file")
}

func TestServerConfig_InvalidStoreIntervalInJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{"store_interval": "not-a-duration"}`)
	os.Args = []string{"test", "-c", path}

	_, err := NewServerConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid store_interval")
}

func TestServerConfig_RestoreFalseFromJSON(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	path := writeTempJSON(t, `{"restore": false}`)
	os.Args = []string{"test", "-c", path}

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.False(t, cfg.Restore)
}

func TestServerConfig_NoConfigFile(t *testing.T) {
	resetFlagsAndArgs()
	clearServerEnv(t)

	cfg, err := NewServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.ListenAddress)
}
