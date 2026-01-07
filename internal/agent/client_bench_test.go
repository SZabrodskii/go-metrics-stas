package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/SZabrodskii/go-metrics-stas/internal/model"
)

func BenchmarkJSONEncode_Single(b *testing.B) {
	v := 123.45
	metric := model.Metrics{
		ID:    "testGauge",
		MType: "gauge",
		Value: &v,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(&metric)
	}
}

func BenchmarkJSONEncode_Batch(b *testing.B) {
	batch := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		v := float64(i)
		batch[i] = model.Metrics{
			ID:    fmt.Sprintf("metric_%d", i),
			MType: "gauge",
			Value: &v,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(&batch)
	}
}

func BenchmarkGzipCompress_Small(b *testing.B) {
	data := bytes.Repeat([]byte("test data "), 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(data)
		_ = zw.Close()
	}
}

func BenchmarkGzipCompress_Medium(b *testing.B) {
	data := bytes.Repeat([]byte("test data for compression "), 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(data)
		_ = zw.Close()
	}
}

func BenchmarkGzipCompress_Large(b *testing.B) {
	data := bytes.Repeat([]byte("test data for compression "), 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(data)
		_ = zw.Close()
	}
}

func BenchmarkFullEncodePipeline_Single(b *testing.B) {
	v := 123.45
	metric := model.Metrics{
		ID:    "testGauge",
		MType: "gauge",
		Value: &v,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var jb bytes.Buffer
		_ = json.NewEncoder(&jb).Encode(&metric)

		var gb bytes.Buffer
		zw := gzip.NewWriter(&gb)
		_, _ = zw.Write(jb.Bytes())
		_ = zw.Close()
	}
}

func BenchmarkFullEncodePipeline_Batch(b *testing.B) {
	batch := make([]model.Metrics, 100)
	for i := 0; i < 100; i++ {
		v := float64(i)
		batch[i] = model.Metrics{
			ID:    fmt.Sprintf("metric_%d", i),
			MType: "gauge",
			Value: &v,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var jb bytes.Buffer
		_ = json.NewEncoder(&jb).Encode(&batch)

		var gb bytes.Buffer
		zw := gzip.NewWriter(&gb)
		_, _ = zw.Write(jb.Bytes())
		_ = zw.Close()
	}
}

func BenchmarkHMACSHA256(b *testing.B) {
	data := bytes.Repeat([]byte("test data for hashing "), 100)
	key := "secret-key-for-testing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hmacSHA256Hex(data, key)
	}
}
