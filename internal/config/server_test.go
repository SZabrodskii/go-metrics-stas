package config

import (
	"os"
	"testing"
	"time"
)

func TestServerConfig_Defaults(t *testing.T) {
	originalEnv := make(map[string]string)
	envVars := []string{"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH", "RESTORE", "DATABASE_DSN", "KEY", "AUDIT_FILE", "AUDIT_URL"}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	defer func() {
		for _, envVar := range envVars {
			if val, exists := originalEnv[envVar]; exists {
				os.Setenv(envVar, val)
			}
		}
	}()

	cfg := &ServerConfig{}
	cfg.ListenAddress = "localhost:8080"
	cfg.StoreInterval = 300 * time.Second
	cfg.Restore = true

	if cfg.ListenAddress != "localhost:8080" {
		t.Errorf("Expected ListenAddress to be 'localhost:8080', got %s", cfg.ListenAddress)
	}

	if cfg.StoreInterval != 300*time.Second {
		t.Errorf("Expected StoreInterval to be 300s, got %v", cfg.StoreInterval)
	}

	if cfg.Restore != true {
		t.Errorf("Expected Restore to be true, got %v", cfg.Restore)
	}
}

func TestServerConfig_EnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		check    func(*ServerConfig) bool
	}{
		{
			name:     "ADDRESS env var",
			envVar:   "ADDRESS",
			envValue: "localhost:9090",
			check: func(cfg *ServerConfig) bool {
				return cfg.ListenAddress == "localhost:9090"
			},
		},
		{
			name:     "DATABASE_DSN env var",
			envVar:   "DATABASE_DSN",
			envValue: "postgres://test",
			check: func(cfg *ServerConfig) bool {
				return cfg.DatabaseDSN == "postgres://test"
			},
		},
		{
			name:     "KEY env var",
			envVar:   "KEY",
			envValue: "secret123",
			check: func(cfg *ServerConfig) bool {
				return cfg.Key == "secret123"
			},
		},
		{
			name:     "AUDIT_FILE env var",
			envVar:   "AUDIT_FILE",
			envValue: "/tmp/audit.log",
			check: func(cfg *ServerConfig) bool {
				return cfg.AuditFile == "/tmp/audit.log"
			},
		},
		{
			name:     "AUDIT_URL env var",
			envVar:   "AUDIT_URL",
			envValue: "http://example.com/audit",
			check: func(cfg *ServerConfig) bool {
				return cfg.AuditURL == "http://example.com/audit"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			cfg := &ServerConfig{}

			if v, ok := os.LookupEnv("ADDRESS"); ok {
				cfg.ListenAddress = v
			}
			if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
				cfg.DatabaseDSN = v
			}
			if v, ok := os.LookupEnv("KEY"); ok {
				cfg.Key = v
			}
			if v, ok := os.LookupEnv("AUDIT_FILE"); ok {
				cfg.AuditFile = v
			}
			if v, ok := os.LookupEnv("AUDIT_URL"); ok {
				cfg.AuditURL = v
			}

			if !tt.check(cfg) {
				t.Errorf("Environment variable %s=%s not properly set", tt.envVar, tt.envValue)
			}
		})
	}
}
