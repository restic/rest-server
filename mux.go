package restserver

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

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

// parseTrustedProxies converts a list of IP addresses and CIDR prefixes into
// prefixes. A plain IP address matches only that address.
func parseTrustedProxies(list []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(list))
	for _, entry := range list {
		entry = strings.TrimSpace(entry)
		if prefix, err := netip.ParsePrefix(entry); err == nil {
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy %q: expected an IP address or CIDR prefix", entry)
		}
		addr = addr.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return prefixes, nil
}

// isTrustedProxy reports whether remoteAddr (in "host:port" form, as found in
// http.Request.RemoteAddr) belongs to one of the configured trusted proxies.
// Addresses that are not IP addresses, such as unix socket peers, are never
// trusted.
func (s *Server) isTrustedProxy(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range s.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// trustedProxyHandler decides, based on the address the request was received
// from, whether the request comes from a trusted proxy. It must run before any
// proxy header is evaluated, as those may rewrite the request's RemoteAddr.
// Requests from trusted proxies are passed to trusted. All other requests are
// passed to untrusted after the proxy authentication header, if any, has been
// removed, so that such requests cannot authenticate via the header.
func (s *Server) trustedProxyHandler(trusted, untrusted http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isTrustedProxy(r.RemoteAddr) {
			trusted.ServeHTTP(w, r)
			return
		}
		if s.ProxyAuthUsername != "" && r.Header.Get(s.ProxyAuthUsername) != "" {
			log.Printf("Ignoring %s header from untrusted address %s", s.ProxyAuthUsername, r.RemoteAddr)
			r.Header.Del(s.ProxyAuthUsername)
		}
		untrusted.ServeHTTP(w, r)
	})
}

func (s *Server) checkAuth(r *http.Request) (username string, ok bool) {
	if s.NoAuth {
		return username, true
	}
	if s.ProxyAuthUsername != "" {
		// with --trusted-proxies, trustedProxyHandler has already removed the
		// header from requests that were not sent by a trusted proxy
		username = r.Header.Get(s.ProxyAuthUsername)
		if username == "" {
			return "", false
		}
		if !validUsernameRegexp.MatchString(username) {
			log.Printf("Rejected invalid username %q in %s header", username, s.ProxyAuthUsername)
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

	if len(server.TrustedProxies) > 0 {
		if server.ProxyAuthUsername == "" && !server.ProxyHeaders {
			return nil, errors.New("--trusted-proxies requires --proxy-auth-username or --proxy-headers")
		}
		var err error
		server.trustedProxies, err = parseTrustedProxies(server.TrustedProxies)
		if err != nil {
			return nil, err
		}
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
		// Only requests from trusted proxies may rewrite the client address.
		// Without --trusted-proxies, every request is treated as trusted.
		withProxyHeaders := handlers.ProxyHeaders(handler)
		if len(server.trustedProxies) > 0 {
			handler = server.trustedProxyHandler(withProxyHeaders, handler)
		} else {
			handler = withProxyHeaders
		}
	} else if len(server.trustedProxies) > 0 {
		handler = server.trustedProxyHandler(handler, handler)
	}
	if server.Log != "" {
		handler = server.logHandler(handler)
	}
	return handler, nil
}
