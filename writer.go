package restserver

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
)

// Writer limits the number of bytes written to the space that is currently
// available as given by the server's MaxRepoSize. This type is safe for use
// by multiple goroutines sharing the same *Server.
type Writer struct {
	io.Writer
	server *Server
}

func (w Writer) Write(p []byte) (n int, err error) {
	if int64(len(p)) > w.server.repoSizeLeft() {
		return 0, fmt.Errorf("repository has reached maximum size (%d bytes)", w.server.conf.MaxRepoSize)
	}
	n, err = w.Writer.Write(p)
	w.server.incrementRepoSize(int64(n))
	return n, err
}

// NewWriter wraps w in a writer that enforces s.MaxRepoSize.
// If there is an error, a status code and the error are returned.
func (s *Server) NewWriter(req *http.Request, w io.Writer) (io.Writer, int, error) {
	// if we haven't yet computed the size of the repo, do so now
	currentSize := atomic.LoadInt64(&s.repoSize)
	if currentSize == 0 {
		initialSize, err := PathSize(s.conf.Path)
		if err != nil {
			return nil, 500, err
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
		if currentSize+contentLen > s.conf.MaxRepoSize {
			err := fmt.Errorf("incoming blob (%d bytes) would exceed maximum size of repository (%d bytes)",
				contentLen, s.conf.MaxRepoSize)
			return nil, http.StatusRequestEntityTooLarge, err
		}
	}

	// since we can't always trust content-length, we will wrap the writer
	// in a custom writer that enforces the size limit during writes
	return Writer{server: s}, 0, nil
}

// SpaceRemaining returns how much space is available in the repo
// according to s.MaxRepoSize. s.repoSize must already be set.
// If there is no limit, -1 is returned.
func (s *Server) repoSizeLeft() int64 {
	if s.conf.MaxRepoSize == 0 {
		return -1
	}
	return s.conf.MaxRepoSize - atomic.LoadInt64(&s.repoSize)
}

// incrementRepoSize increments the current repo size (which
// must already be initialized).
func (s *Server) incrementRepoSize(by int64) {
	atomic.AddInt64(&s.repoSize, by)
}
