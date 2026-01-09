package audit

import (
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestCreateAuditEvent(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		metrics    []string
		expectedIP string
	}{
		{
			name:       "Basic IP from RemoteAddr",
			remoteAddr: "192.168.1.100:12345",
			metrics:    []string{"TestMetric"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "IP from X-Forwarded-For",
			remoteAddr: "192.168.1.100:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18"},
			metrics:    []string{"Alloc", "Frees"},
			expectedIP: "203.0.113.195",
		},
		{
			name:       "IP from X-Real-IP",
			remoteAddr: "192.168.1.100:12345",
			headers:    map[string]string{"X-Real-IP": "198.51.100.42"},
			metrics:    []string{"TestMetric"},
			expectedIP: "198.51.100.42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			event := CreateAuditEvent(req, tt.metrics)

			if event.IPAddress != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, event.IPAddress)
			}

			if len(event.Metrics) != len(tt.metrics) {
				t.Errorf("Expected %d metrics, got %d", len(tt.metrics), len(event.Metrics))
			}

			if event.Timestamp == 0 {
				t.Error("Expected non-zero timestamp")
			}
		})
	}
}

func TestPublisher(t *testing.T) {
	logger := zaptest.NewLogger(t)
	publisher := NewPublisher(logger)

	auditor := &mockAuditor{id: "test"}

	publisher.Subscribe(auditor)
	if _, exists := publisher.observers["test"]; !exists {
		t.Error("Expected auditor to be subscribed")
	}

	publisher.Unsubscribe(auditor)
	if _, exists := publisher.observers["test"]; exists {
		t.Error("Expected auditor to be unsubscribed")
	}
}

type mockAuditor struct {
	id string
}

func (m *mockAuditor) Update(event AuditEvent) error {
	return nil
}

func (m *mockAuditor) GetID() string {
	return m.id
}
