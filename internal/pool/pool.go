// Package pool предоставляет пулы объектов для переиспользования буферов и gzip writer/reader.
// Использование пулов снижает нагрузку на GC и уменьшает количество аллокаций.
package pool

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"
)

// BufferPool — пул для переиспользования bytes.Buffer.
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer возвращает буфер из пула (сброшенный в начальное состояние).
func GetBuffer() *bytes.Buffer {
	buf := BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer возвращает буфер в пул.
// Буферы с ёмкостью > 64KB не возвращаются в пул во избежание утечки памяти.
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

// GzipWriterPool — пул для переиспользования gzip.Writer.
var GzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// GetGzipWriter возвращает gzip.Writer из пула, настроенный на указанный writer.
func GetGzipWriter(w io.Writer) *gzip.Writer {
	zw := GzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(w)
	return zw
}

// PutGzipWriter возвращает gzip.Writer в пул.
// Закрывает writer перед возвратом.
func PutGzipWriter(zw *gzip.Writer) {
	if zw == nil {
		return
	}

	_ = zw.Close()
	GzipWriterPool.Put(zw)
}

// GzipReaderPool — пул для переиспользования gzip.Reader.
var GzipReaderPool = sync.Pool{}

// GetGzipReader возвращает gzip.Reader из пула, настроенный на указанный reader.
// Если пул пуст, создаёт новый gzip.Reader.
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

// PutGzipReader возвращает gzip.Reader в пул.
// Закрывает reader перед возвратом.
func PutGzipReader(zr *gzip.Reader) {
	if zr == nil {
		return
	}
	_ = zr.Close()
	GzipReaderPool.Put(zr)
}
