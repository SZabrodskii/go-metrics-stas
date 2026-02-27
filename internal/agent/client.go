package agent

import (
	"bytes"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	appcrypto "github.com/SZabrodskii/go-metrics-stas/internal/crypto"
	"github.com/SZabrodskii/go-metrics-stas/internal/model"
	"github.com/SZabrodskii/go-metrics-stas/internal/pool"
)

var (
	retrySchedule = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	ErrEncodeJSON      = errors.New("encode metric json")
	ErrCompressMetrics = errors.New("compress metrics")
	ErrEncryptBody     = errors.New("encrypt body")
	ErrBuildRequest    = errors.New("build request")
	ErrBatchNotFound   = errors.New("batch endpoint was not found")
)

type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return "unexpected server status: " + strconv.Itoa(e.Code)
}

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
			select {
			case <-t.C:
			case <-req.Context().Done():
				t.Stop()
				return nil, req.Context().Err()
			}
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
	key       string
	publicKey *rsa.PublicKey
	localIP   string
}

func resolveLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	return addr.IP.String()
}

func newMetricsClient(serverURL string, key string, publicKey *rsa.PublicKey) *metricsClient {
	return &metricsClient{
		serverURL: serverURL,
		client: &retryHTTPClient{
			base:          &http.Client{Timeout: 5 * time.Second},
			retrySchedule: retrySchedule,
		},
		key:       key,
		publicKey: publicKey,
		localIP:   resolveLocalIP(),
	}
}

func hmacSHA256Hex(b []byte, key string) string {
	if key == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(b)

	return hex.EncodeToString(mac.Sum(nil))
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

	jb := pool.GetBuffer()
	defer pool.PutBuffer(jb)

	if err := json.NewEncoder(jb).Encode(&payload); err != nil {
		return errors.Join(ErrEncodeJSON, err)
	}

	gb := pool.GetBuffer()
	defer pool.PutBuffer(gb)

	zw := pool.GetGzipWriter(gb)
	if _, err := zw.Write(jb.Bytes()); err != nil {
		pool.PutGzipWriter(zw)
		return errors.Join(ErrCompressMetrics, err)
	}
	pool.PutGzipWriter(zw)

	bodyBytes := make([]byte, gb.Len())
	copy(bodyBytes, gb.Bytes())

	bodyBytes, err := appcrypto.Encrypt(bodyBytes, mc.publicKey)
	if err != nil {
		return errors.Join(ErrEncryptBody, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return errors.Join(ErrBuildRequest, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if mc.localIP != "" {
		req.Header.Set("X-Real-IP", mc.localIP)
	}
	if mc.key != "" {
		req.Header.Set("HashSHA256", hmacSHA256Hex(jb.Bytes(), mc.key))
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &StatusError{Code: resp.StatusCode}
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

	jb := pool.GetBuffer()
	defer pool.PutBuffer(jb)

	if err := json.NewEncoder(jb).Encode(&metrics); err != nil {
		return errors.Join(ErrEncodeJSON, err)
	}

	gb := pool.GetBuffer()
	defer pool.PutBuffer(gb)

	zw := pool.GetGzipWriter(gb)
	if _, err := zw.Write(jb.Bytes()); err != nil {
		pool.PutGzipWriter(zw)
		return errors.Join(ErrCompressMetrics, err)
	}
	pool.PutGzipWriter(zw)

	gzBytes := make([]byte, gb.Len())
	copy(gzBytes, gb.Bytes())

	gzBytes, err := appcrypto.Encrypt(gzBytes, mc.publicKey)
	if err != nil {
		return errors.Join(ErrEncryptBody, err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(gzBytes))
	if err != nil {
		return errors.Join(ErrBuildRequest, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if mc.localIP != "" {
		req.Header.Set("X-Real-IP", mc.localIP)
	}
	if mc.key != "" {
		req.Header.Set("HashSHA256", hmacSHA256Hex(jb.Bytes(), mc.key))
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(gzBytes)), nil
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrBatchNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return &StatusError{Code: resp.StatusCode}
	}

	return nil
}
