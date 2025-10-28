package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

type metricsClient struct {
	serverURL string
	client    *http.Client
}

func newMetricsClient(serverURL string) *metricsClient {
	return &metricsClient{
		serverURL: serverURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

var retrySchedule = []time.Duration{1 * time.Second, 30 * time.Second, 5 * time.Second}

func shouldRetryHTTP(resp *http.Response, err error) bool {
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return true
		}
		return true
	}

	if resp != nil && resp.StatusCode >= 500 {
		return true
	}
	return false
}

func (mc *metricsClient) doWithRetry(buildReq func() (*http.Request, error)) (*http.Response, error) {
	var resp *http.Response
	var err error

	attempts := len(retrySchedule) + 1
	for i := 0; i < attempts; i++ {
		req, reqErr := buildReq()
		if reqErr != nil {
			return nil, reqErr
		}

		resp, err = mc.client.Do(req)
		if !shouldRetryHTTP(resp, err) || i == len(retrySchedule) {
			return resp, err
		}

		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		time.Sleep(retrySchedule[i])
	}
	return resp, err
}

func (mc *metricsClient) SendMetric(metric model.Metrics) error {
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

	buildReq := func() (*http.Request, error) {
		var jb bytes.Buffer
		if err := json.NewEncoder(&jb).Encode(&payload); err != nil {
			return nil, fmt.Errorf("could not encode metrics to json: %v", err)
		}

		var gb bytes.Buffer
		zw := gzip.NewWriter(&gb)
		if _, err := zw.Write(jb.Bytes()); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("could not compress metrics: %v", err)
		}
		if err := zw.Close(); err != nil {
			return nil, fmt.Errorf("could not close gzip writer: %v", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(gb.Bytes()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		return req, nil
	}

	resp, err := mc.doWithRetry(buildReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var ack model.Metrics
	_ = json.NewDecoder(resp.Body).Decode(&ack)

	return nil
}

func (mc *metricsClient) SendMetrics(metrics map[string]model.Metrics) error {
	for _, metric := range metrics {
		if err := mc.SendMetric(metric); err != nil {
			log.Printf("Failed to send metric %s: %v", metric.ID, err)
		}
	}
	return nil
}

func (mc *metricsClient) SendBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	url := fmt.Sprintf("%s/updates", mc.serverURL)

	buildReq := func() (*http.Request, error) {
		var jb bytes.Buffer
		if err := json.NewEncoder(&jb).Encode(&metrics); err != nil {
			return nil, fmt.Errorf("could not encode metrics to json: %v", err)
		}

		var gb bytes.Buffer
		zw := gzip.NewWriter(&gb)
		if _, err := zw.Write(jb.Bytes()); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("could not compress metrics: %v", err)
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(gb.Bytes()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		return req, nil
	}

	resp, err := mc.doWithRetry(buildReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("batch endpoint was not found")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
