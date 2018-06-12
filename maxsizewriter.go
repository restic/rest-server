package restserver

import (
	"fmt"
	"io"
	"sync/atomic"
)

// maxSizeWriter limits the number of bytes written
// to the space that is currently available as given by
// the server's MaxRepoSize. This type is safe for use
// by multiple goroutines sharing the same *Server.
type maxSizeWriter struct {
	io.Writer
	server *Server
}

func (w maxSizeWriter) Write(p []byte) (n int, err error) {
	if int64(len(p)) > w.spaceRemaining() {
		return 0, fmt.Errorf("repository has reached maximum size (%d bytes)", w.server.MaxRepoSize)
	}
	n, err = w.Writer.Write(p)
	w.incrementUsage(int64(n))
	return n, err
}

func (w maxSizeWriter) incrementUsage(by int64) {
	atomic.AddInt64(&w.server.repoSize, by)
}

func (w maxSizeWriter) spaceRemaining() int64 {
	maxSize := w.server.MaxRepoSize
	currentSize := atomic.LoadInt64(&w.server.repoSize)
	return maxSize - currentSize
}
