package agent

import (
	"fmt"
	"net/http"
	"log"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
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
	var url string
	
	if metric.MType == model.Gauge && metric.Value != nil {
		url = fmt.Sprintf("%s/update/gauge/%s/%f", mc.serverURL, metric.ID, *metric.Value)
	} else if metric.MType == model.Counter && metric.Delta != nil {
		url = fmt.Sprintf("%s/update/counter/%s/%d", mc.serverURL, metric.ID, *metric.Delta)
	} else {
		return fmt.Errorf("invalid metric type or value")
	}
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := mc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}
	
	log.Printf("Sent metric %s: %s", metric.ID, url)
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