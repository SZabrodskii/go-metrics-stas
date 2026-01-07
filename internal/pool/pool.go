package pool

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
)

var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func GetBuffer() *bytes.Buffer {
	buf := BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	BufferPool.Put(buf)
}

var GzipWriterPool = sync.Pool{
	New: func() interface{} {
		// Create a writer with nil target, will be reset before use
		return gzip.NewWriter(io.Discard)
	},
}

func GetGzipWriter(w io.Writer) *gzip.Writer {
	zw := GzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(w)
	return zw
}

func PutGzipWriter(zw *gzip.Writer) {
	if zw == nil {
		return
	}

	_ = zw.Close()
	GzipWriterPool.Put(zw)
}

var GzipReaderPool = sync.Pool{}

func GetGzipReader(r io.Reader) (*gzip.Reader, error) {
	if v := GzipReaderPool.Get(); v != nil {
		zr := v.(*gzip.Reader)
		if err := zr.Reset(r); err != nil {
			return nil, err
		}
		return zr, nil
	}
	return gzip.NewReader(r)
}

func PutGzipReader(zr *gzip.Reader) {
	if zr == nil {
		return
	}
	_ = zr.Close()
	GzipReaderPool.Put(zr)
}
