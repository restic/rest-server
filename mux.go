package restserver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/restic/rest-server/quota"
)

func (s *Server) debugHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s", r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
}

// closeOnUnreadBody makes the server close an HTTP/1.x connection instead of
// draining a request body that the handler did not read.
//
// To keep a connection alive after a handler replied without reading the
// request body, net/http reads up to 256 KiB of the remaining body: before
// sending the response, and again after the handler has returned. Without
// http.Server.ReadTimeout, which cannot be set as uploads of large files
// legitimately take long, these reads have no deadline. A client that stops
// sending the body could thus hold the connection open forever, for example
// on a request that was rejected because it lacks valid credentials.
//
// Two measures prevent this:
//   - "Connection: close" makes net/http skip draining the body before it
//     sends the response. The header is set up front and removed once the
//     handler has read the body to the end, so that successful uploads keep
//     the connection alive.
//   - If the handler returns without having read the body to the end, the
//     read deadline is set to now, so that draining the body afterwards
//     fails immediately and net/http closes the connection.
func closeOnUnreadBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HTTP/2 does not drain request bodies, and on HTTP/2 a
		// "Connection: close" header would shut down the whole connection.
		// A ContentLength of zero means that the request has no body.
		if r.ProtoMajor != 1 || r.ContentLength == 0 {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Connection", "close")
		body := &keepAliveOnEOF{ReadCloser: r.Body, header: w.Header()}
		r.Body = body
		next.ServeHTTP(w, r)

		if !body.sawEOF {
			// The connection is not reused anyway, as the header told the
			// client. Fails with http.ErrNotSupported for response writers
			// that are not backed by a connection, e.g. in tests.
			_ = http.NewResponseController(w).SetReadDeadline(time.Now())
		}
	})
}

// keepAliveOnEOF removes the "Connection" header from the response once the
// request body has been read to the end.
type keepAliveOnEOF struct {
	io.ReadCloser
	header http.Header
	sawEOF bool
}

func (b *keepAliveOnEOF) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF && !b.sawEOF {
		b.sawEOF = true
		b.header.Del("Connection")
	}
	return n, err
}

func (s *Server) logHandler(next http.Handler) http.Handler {
	var accessLog io.Writer

	if s.Log == "-" {
		accessLog = os.Stdout
	} else {
		var err error
		accessLog, err = os.OpenFile(s.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}

	return handlers.CombinedLoggingHandler(accessLog, next)
}

func (s *Server) checkAuth(r *http.Request) (username string, ok bool) {
	if s.NoAuth {
		return username, true
	}
	if s.ProxyAuthUsername != "" {
		username = r.Header.Get(s.ProxyAuthUsername)
		if username == "" {
			return "", false
		}
	} else {
		var password string
		username, password, ok = r.BasicAuth()
		if !ok || !s.htpasswdFile.Validate(username, password) {
			return "", false
		}
	}
	return username, true
}

func (s *Server) writeUnauthorized(w http.ResponseWriter) {
	if !s.NoAuth && s.ProxyAuthUsername == "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="rest-server"`)
	}
	httpDefaultError(w, http.StatusUnauthorized)
}

func (s *Server) wrapMetricsAuth(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := s.checkAuth(r)
		if !ok {
			s.writeUnauthorized(w)
			return
		}
		if s.PrivateRepos && username != "metrics" {
			httpDefaultError(w, http.StatusUnauthorized)
			return
		}
		f(w, r)
	}
}

// NewHandler returns the master HTTP multiplexer/router.
func NewHandler(server *Server) (http.Handler, error) {
	if !server.NoAuth && server.ProxyAuthUsername == "" {
		var err error
		if server.HtpasswdPath == "" {
			server.HtpasswdPath = filepath.Join(server.Path, ".htpasswd")
		}
		server.htpasswdFile, err = NewHtpasswdFromFile(server.HtpasswdPath)
		if err != nil {
			return nil, fmt.Errorf("cannot load %s (use --no-auth to disable): %v", server.HtpasswdPath, err)
		}
		log.Printf("Loaded htpasswd file %s", server.HtpasswdPath)
	}

	const GiB = 1024 * 1024 * 1024

	if server.MaxRepoSize > 0 {
		log.Printf("Initializing quota (can take a while)...")
		qm, err := quota.New(server.Path, server.MaxRepoSize)
		if err != nil {
			return nil, err
		}
		server.quotaManager = qm
		log.Printf("Quota initialized, currently using %.2f GiB", float64(qm.SpaceUsed())/GiB)
	}

	mux := http.NewServeMux()
	if server.Prometheus {
		if server.PrometheusNoAuth {
			mux.Handle("/metrics", promhttp.Handler())
		} else {
			mux.HandleFunc("/metrics", server.wrapMetricsAuth(promhttp.Handler().ServeHTTP))
		}
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", server)

	var handler http.Handler = mux
	if server.Debug {
		handler = server.debugHandler(handler)
	}
	if server.ProxyHeaders {
		handler = handlers.ProxyHeaders(handler)
	}
	if server.Log != "" {
		handler = server.logHandler(handler)
	}
	// Outermost, so that it also covers requests answered by the mux itself,
	// such as redirects.
	handler = closeOnUnreadBody(handler)
	return handler, nil
}
