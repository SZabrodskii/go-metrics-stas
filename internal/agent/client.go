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

var retrySchedule = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type retryHTTPClient struct {
	base          *http.Client
	retrySchedule []time.Duration
}

func shouldRetryHTTP(resp *http.Response, err error) bool {
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return true
		}
		return true
	}

	return resp != nil && resp.StatusCode >= 500
}

func (c *retryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	resetBody := func(r *http.Request) error {
		if r.GetBody == nil {
			return nil
		}
		rc, gerr := r.GetBody()
		if gerr != nil {
			return gerr
		}
		if r.Body != nil {
			_ = r.Body.Close()
		}
		r.Body = rc
		return nil
	}

	attempts := len(c.retrySchedule) + 1
	for i := 0; i < attempts; i++ {
		if i > 0 {
			t := time.NewTimer(c.retrySchedule[i-1])
			<-t.C
			t.Stop()
			_ = resetBody(req)
		}

		resp, err = c.base.Do(req)
		if !shouldRetryHTTP(resp, err) || i == len(c.retrySchedule) {
			return resp, err

		}

		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}
	return resp, err
}

type metricsClient struct {
	serverURL string
	client    httpDoer
}

func newMetricsClient(serverURL string) *metricsClient {
	return &metricsClient{
		serverURL: serverURL,
		client: &retryHTTPClient{
			base:          &http.Client{Timeout: 5 * time.Second},
			retrySchedule: retrySchedule,
		},
	}
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

	var jb bytes.Buffer
	if err := json.NewEncoder(&jb).Encode(&payload); err != nil {
		return fmt.Errorf("encode metric json: %w", err)
	}

	var gb bytes.Buffer
	zw := gzip.NewWriter(&gb)
	if _, err := zw.Write(jb.Bytes()); err != nil {
		_ = zw.Close()
		return fmt.Errorf("could not compress metrics: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("could not close gzip writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(gb.Bytes()))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	bodyBytes := gb.Bytes()
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

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

	var jb bytes.Buffer
	if err := json.NewEncoder(&jb).Encode(&metrics); err != nil {
		return fmt.Errorf("could not encode metrics to json: %v", err)
	}

	var gb bytes.Buffer
	zw := gzip.NewWriter(&gb)
	if _, err := zw.Write(jb.Bytes()); err != nil {
		_ = zw.Close()
		return fmt.Errorf("could not compress metrics: %v", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(gb.Bytes()))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	batchBytes := jb.Bytes()
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(batchBytes)), nil
	}

	resp, err := mc.client.Do(req)
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
