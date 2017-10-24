package datacounter

import (
	"io"
	"sync/atomic"
)

// ReaderCounter is counter for io.Reader
type ReaderCounter struct {
	io.Reader
	count  uint64
	reader io.Reader
}

// NewReaderCounter function for create new ReaderCounter
func NewReaderCounter(r io.Reader) *ReaderCounter {
	return &ReaderCounter{
		reader: r,
	}
}

func (counter *ReaderCounter) Read(buf []byte) (int, error) {
	n, err := counter.reader.Read(buf)
	atomic.AddUint64(&counter.count, uint64(n))
	return n, err
}

// Count function return counted bytes
func (counter *ReaderCounter) Count() uint64 {
	return atomic.LoadUint64(&counter.count)
}
