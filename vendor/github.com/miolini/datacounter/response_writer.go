package datacounter

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// ResponseWriterCounter is counter for http.ResponseWriter
type ResponseWriterCounter struct {
	http.ResponseWriter
	count   uint64
	started time.Time
	writer  http.ResponseWriter
}

// NewResponseWriterCounter function create new ResponseWriterCounter
func NewResponseWriterCounter(rw http.ResponseWriter) *ResponseWriterCounter {
	return &ResponseWriterCounter{
		writer: rw,
		started: time.Now(),
	}
}

func (counter *ResponseWriterCounter) Write(buf []byte) (int, error) {
	n, err := counter.writer.Write(buf)
	atomic.AddUint64(&counter.count, uint64(n))
	return n, err
}

func (counter *ResponseWriterCounter) Header() http.Header {
	return counter.writer.Header()
}

func (counter *ResponseWriterCounter) WriteHeader(statusCode int) {
	counter.Header().Set("X-Runtime", fmt.Sprintf("%.6f", time.Since(counter.started).Seconds()))
	counter.writer.WriteHeader(statusCode)
}

func (counter *ResponseWriterCounter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return counter.writer.(http.Hijacker).Hijack()
}

// Count function return counted bytes
func (counter *ResponseWriterCounter) Count() uint64 {
	return atomic.LoadUint64(&counter.count)
}

func (counter *ResponseWriterCounter) Started() time.Time {
	return counter.started
}
