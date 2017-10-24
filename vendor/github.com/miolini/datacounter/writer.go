package datacounter

import (
	"io"
	"sync/atomic"
)

// WriterCounter is counter for io.Writer
type WriterCounter struct {
	io.Writer
	count  uint64
	writer io.Writer
}

// NewWriterCounter function create new WriterCounter
func NewWriterCounter(w io.Writer) *WriterCounter {
	return &WriterCounter{
		writer: w,
	}
}

func (counter *WriterCounter) Write(buf []byte) (int, error) {
	n, err := counter.writer.Write(buf)
	atomic.AddUint64(&counter.count, uint64(n))
	return n, err
}

// Count function return counted bytes
func (counter *WriterCounter) Count() uint64 {
	return atomic.LoadUint64(&counter.count)
}
