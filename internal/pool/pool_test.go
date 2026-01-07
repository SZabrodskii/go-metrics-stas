package pool

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestBufferPool(t *testing.T) {
	buf := GetBuffer()
	if buf == nil {
		t.Fatal("expected non-nil buffer")
	}

	buf.WriteString("test data")
	if buf.String() != "test data" {
		t.Errorf("expected 'test data', got %q", buf.String())
	}

	PutBuffer(buf)

	buf2 := GetBuffer()
	if buf2.Len() != 0 {
		t.Error("buffer should be reset after getting from pool")
	}
}

func TestBufferPool_LargeBuffer(t *testing.T) {
	buf := GetBuffer()

	largeData := strings.Repeat("x", 100*1024)
	buf.WriteString(largeData)

	PutBuffer(buf)

	buf2 := GetBuffer()
	if buf2.Cap() > 64*1024 {
		t.Error("should not reuse large buffers")
	}
}

func TestGzipWriterPool(t *testing.T) {
	var buf bytes.Buffer

	zw := GetGzipWriter(&buf)
	if zw == nil {
		t.Fatal("expected non-nil gzip writer")
	}

	_, err := zw.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	PutGzipWriter(zw)

	var buf2 bytes.Buffer
	zw2 := GetGzipWriter(&buf2)
	if zw2 == nil {
		t.Fatal("expected non-nil gzip writer")
	}

	_, err = zw2.Write([]byte("more data"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	PutGzipWriter(zw2)
}

func TestGzipReaderPool(t *testing.T) {
	var compressed bytes.Buffer
	zw := GetGzipWriter(&compressed)
	_, _ = zw.Write([]byte("test data"))
	PutGzipWriter(zw)

	zr, err := GetGzipReader(&compressed)
	if err != nil {
		t.Fatalf("unexpected error creating reader: %v", err)
	}

	var decompressed bytes.Buffer
	_, err = decompressed.ReadFrom(zr)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}

	if decompressed.String() != "test data" {
		t.Errorf("expected 'test data', got %q", decompressed.String())
	}

	PutGzipReader(zr)
}

func TestNilHandling(t *testing.T) {
	PutBuffer(nil)
	PutGzipWriter(nil)
	PutGzipReader(nil)
}

func BenchmarkBufferPool_WithPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		buf.WriteString("test data for benchmarking")
		PutBuffer(buf)
	}
}

func BenchmarkBufferPool_WithoutPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.WriteString("test data for benchmarking")
	}
}

func BenchmarkGzipWriterPool_WithPool(b *testing.B) {
	b.ReportAllocs()
	data := []byte("test data for compression benchmarking")
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		zw := GetGzipWriter(buf)
		_, _ = zw.Write(data)
		PutGzipWriter(zw)
		PutBuffer(buf)
	}
}

func BenchmarkGzipWriterPool_WithoutPool(b *testing.B) {
	b.ReportAllocs()
	data := []byte("test data for compression benchmarking")
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(data)
		_ = zw.Close()
	}
}
