package restserver

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	if int64(len(p)) > w.server.repoSpaceRemaining() {
		return 0, fmt.Errorf("repository has reached maximum size (%d bytes)", w.server.MaxRepoSize)
	}
	n, err = w.Writer.Write(p)
	w.server.incrementRepoSpaceUsage(int64(n))
	return n, err
}

// maxSizeWriter wraps w in a writer that enforces s.MaxRepoSize.
// If there is an error, a status code and the error are returned.
func (s *Server) maxSizeWriter(req *http.Request, w io.Writer) (io.Writer, int, error) {
	// if we haven't yet computed the size of the repo, do so now
	currentSize := atomic.LoadInt64(&s.repoSize)
	if currentSize == 0 {
		initialSize, err := tallySize(s.Path)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		atomic.StoreInt64(&s.repoSize, initialSize)
		currentSize = initialSize
	}

	// if content-length is set and is trustworthy, we can save some time
	// and issue a polite error if it declares a size that's too big; since
	// we expect the vast majority of clients will be honest, so this check
	// can only help save time
	if contentLenStr := req.Header.Get("Content-Length"); contentLenStr != "" {
		contentLen, err := strconv.ParseInt(contentLenStr, 10, 64)
		if err != nil {
			return nil, http.StatusLengthRequired, err
		}
		if currentSize+contentLen > s.MaxRepoSize {
			err := fmt.Errorf("incoming blob (%d bytes) would exceed maximum size of repository (%d bytes)",
				contentLen, s.MaxRepoSize)
			return nil, http.StatusRequestEntityTooLarge, err
		}
	}

	// since we can't always trust content-length, we will wrap the writer
	// in a custom writer that enforces the size limit during writes
	return maxSizeWriter{Writer: w, server: s}, 0, nil
}

// repoSpaceRemaining returns how much space is available in the repo
// according to s.MaxRepoSize. s.repoSize must already be set.
// If there is no limit, -1 is returned.
func (s *Server) repoSpaceRemaining() int64 {
	if s.MaxRepoSize == 0 {
		return -1
	}
	maxSize := s.MaxRepoSize
	currentSize := atomic.LoadInt64(&s.repoSize)
	return maxSize - currentSize
}

// incrementRepoSpaceUsage increments the current repo size (which
// must already be initialized).
func (s *Server) incrementRepoSpaceUsage(by int64) {
	atomic.AddInt64(&s.repoSize, by)
}
