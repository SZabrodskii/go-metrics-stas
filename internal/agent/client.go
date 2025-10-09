package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"log"
	"net/http"
)

type MetricsClient struct {
	serverURL string
	client    *http.Client
}

func NewMetricsClient(serverURL string) *MetricsClient {
	return &MetricsClient{
		serverURL: serverURL,
		client:    &http.Client{},
	}
}

func (mc *MetricsClient) SendMetric(metric model.Metrics) error {
	url := fmt.Sprintf("%s/update", mc.serverURL)

	var payload struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Delta *int64   `json:"delta,omitempty"`
		Value *float64 `json:"value,omitempty"`
	}

	payload.ID = metric.ID
	payload.MType = metric.MType
	payload.Delta = metric.Delta
	payload.Value = metric.Value

	body, err := json.Marshal(&payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var ack model.Metrics
	_ = json.NewDecoder(resp.Body).Decode(&ack)

	log.Printf("Sent metric %s (%s) via JSON", metric.ID, metric.MType)
	return nil
}

func (mc *MetricsClient) SendMetrics(metrics map[string]model.Metrics) error {
	for _, metric := range metrics {
		if err := mc.SendMetric(metric); err != nil {
			log.Printf("Failed to send metric %s: %v", metric.ID, err)
		}
	}
	return nil
}
