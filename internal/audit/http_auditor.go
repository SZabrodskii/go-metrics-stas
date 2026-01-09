package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type HTTPAuditor struct {
	url    string
	client *http.Client
	logger *zap.Logger
}

func NewHTTPAuditor(url string, logger *zap.Logger) *HTTPAuditor {
	return &HTTPAuditor{
		url: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (ha *HTTPAuditor) Update(event AuditEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		ha.logger.Error("failed to marshal audit event for HTTP", zap.Error(err))
		return err
	}

	req, err := http.NewRequest(http.MethodPost, ha.url, bytes.NewBuffer(eventJSON))
	if err != nil {
		ha.logger.Error("failed to create HTTP request for audit",
			zap.String("url", ha.url),
			zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "metrics-audit-client/1.0")

	resp, err := ha.client.Do(req)
	if err != nil {
		ha.logger.Error("failed to send audit event via HTTP",
			zap.String("url", ha.url),
			zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("HTTP audit request failed with status: %d %s",
			resp.StatusCode, resp.Status)
		ha.logger.Error("HTTP audit request failed",
			zap.String("url", ha.url),
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status))
		return err
	}

	ha.logger.Debug("audit event sent via HTTP",
		zap.String("url", ha.url),
		zap.Int("metrics_count", len(event.Metrics)),
		zap.String("ip", event.IPAddress),
		zap.Int("status_code", resp.StatusCode))

	return nil
}

func (ha *HTTPAuditor) GetID() string {
	return "http_auditor_" + ha.url
}
