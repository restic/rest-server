package restserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name           string
		server         *Server
		requestHeaders map[string]string
		basicAuth      bool
		basicUser      string
		basicPassword  string
		expectedUser   string
		expectedOk     bool
	}{
		{
			name: "NoAuth enabled",
			server: &Server{
				NoAuth: true,
			},
			expectedOk: true,
		},
		{
			name: "Proxy Auth successful",
			server: &Server{
				ProxyAuthUsername: "X-Remote-User",
			},
			requestHeaders: map[string]string{
				"X-Remote-User": "restic",
			},
			expectedUser: "restic",
			expectedOk:   true,
		},
		{
			name: "Proxy Auth empty header",
			server: &Server{
				ProxyAuthUsername: "X-Remote-User",
			},
			requestHeaders: map[string]string{
				"X-Remote-User": "",
			},
			expectedOk: false,
		},
		{
			name: "Proxy Auth missing header",
			server: &Server{
				ProxyAuthUsername: "X-Remote-User",
			},
			expectedOk: false,
		},
		{
			name:   "Proxy Auth send but not enabled",
			server: &Server{},
			requestHeaders: map[string]string{
				"X-Remote-User": "restic",
			},
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for header, value := range tt.requestHeaders {
				req.Header.Set(header, value)
			}
			if tt.basicAuth {
				req.SetBasicAuth(tt.basicUser, tt.basicPassword)
			}

			username, ok := tt.server.checkAuth(req)
			if username != tt.expectedUser || ok != tt.expectedOk {
				t.Errorf("expected (%v, %v), got (%v, %v)", tt.expectedUser, tt.expectedOk, username, ok)
			}
		})
	}
}

func TestHealthz(t *testing.T) {
	handler, err := NewHandler(&Server{NoAuth: true})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestUnauthorizedResponsesIncludeBasicChallenge(t *testing.T) {
	const challenge = `Basic realm="rest-server"`

	tests := []struct {
		name          string
		server        *Server
		metrics       bool
		wantChallenge string
	}{
		{
			name:          "server Basic authentication",
			server:        &Server{},
			wantChallenge: challenge,
		},
		{
			name:   "server proxy authentication",
			server: &Server{ProxyAuthUsername: "X-Remote-User"},
		},
		{
			name:          "metrics Basic authentication",
			server:        &Server{},
			metrics:       true,
			wantChallenge: challenge,
		},
		{
			name:    "metrics proxy authentication",
			server:  &Server{ProxyAuthUsername: "X-Remote-User"},
			metrics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handler http.HandlerFunc
			if tt.metrics {
				handler = tt.server.wrapMetricsAuth(func(http.ResponseWriter, *http.Request) {
					t.Fatal("handler should not be called for an unauthorized request")
				})
			} else {
				handler = tt.server.ServeHTTP
			}

			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != tt.wantChallenge {
				t.Errorf("WWW-Authenticate = %q, want %q", got, tt.wantChallenge)
			}
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	for _, test := range []struct {
		input   []string
		want    []string
		wantErr bool
	}{
		{input: []string{"10.0.0.1"}, want: []string{"10.0.0.1/32"}},
		{input: []string{"::1"}, want: []string{"::1/128"}},
		{input: []string{"::ffff:10.0.0.1"}, want: []string{"10.0.0.1/32"}},
		{input: []string{"10.1.2.3/8", " fd00::/64 "}, want: []string{"10.0.0.0/8", "fd00::/64"}},
		{input: []string{"proxy.example.com"}, wantErr: true},
		{input: []string{"10.0.0.1:8080"}, wantErr: true},
		{input: []string{"10.0.0.0/33"}, wantErr: true},
		{input: []string{""}, wantErr: true},
	} {
		prefixes, err := parseTrustedProxies(test.input)
		if test.wantErr {
			if err == nil {
				t.Errorf("parseTrustedProxies(%q): expected error, got %v", test.input, prefixes)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseTrustedProxies(%q): unexpected error %v", test.input, err)
			continue
		}
		got := make([]string, 0, len(prefixes))
		for _, prefix := range prefixes {
			got = append(got, prefix.String())
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("parseTrustedProxies(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestNewHandlerTrustedProxies(t *testing.T) {
	for _, test := range []struct {
		name    string
		conf    *Server
		wantErr bool
	}{
		{"with proxy auth", &Server{ProxyAuthUsername: "X-Forwarded-User", TrustedProxies: []string{"10.0.0.0/8"}}, false},
		{"with proxy headers", &Server{NoAuth: true, ProxyHeaders: true, TrustedProxies: []string{"10.0.0.1"}}, false},
		{"without proxy options", &Server{NoAuth: true, TrustedProxies: []string{"10.0.0.1"}}, true},
		{"invalid entry", &Server{ProxyAuthUsername: "X-Forwarded-User", TrustedProxies: []string{"proxy"}}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.conf.Path = t.TempDir()
			_, err := NewHandler(test.conf)
			if test.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !test.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTrustedProxyHandler(t *testing.T) {
	s := &Server{ProxyAuthUsername: "X-Forwarded-User"}
	var err error
	s.trustedProxies, err = parseTrustedProxies([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}

	var got, username string
	handler := s.trustedProxyHandler(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = "trusted"
			username = r.Header.Get("X-Forwarded-User")
		}),
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = "untrusted"
			username = r.Header.Get("X-Forwarded-User")
		}),
	)

	for _, test := range []struct {
		remoteAddr string
		want       string
	}{
		{"10.0.0.1:1234", "trusted"},
		{"[::ffff:10.0.0.1]:1234", "trusted"},
		{"192.0.2.1:1234", "untrusted"},
		{"[::1]:1234", "untrusted"},
		{"10.0.0.1", "untrusted"}, // not in host:port form
		{"@", "untrusted"},        // unix socket peer
		{"", "untrusted"},
	} {
		got, username = "", ""
		req := proxyAuthRequest(t, "GET", "/", "alice", test.remoteAddr, nil)
		checkRequest(t, handler.ServeHTTP, req, nil)
		if got != test.want {
			t.Errorf("remote address %q: request went to %s handler, want %s", test.remoteAddr, got, test.want)
		}
		wantUsername := ""
		if test.want == "trusted" {
			wantUsername = "alice"
		}
		if username != wantUsername {
			t.Errorf("remote address %q: username header is %q, want %q", test.remoteAddr, username, wantUsername)
		}
	}
}

// proxyAuthRequest returns a request that carries username in the
// X-Forwarded-User header (if not empty) and was sent from remoteAddr.
func proxyAuthRequest(t testing.TB, method, path, username, remoteAddr string, body io.Reader) *http.Request {
	req := newRequest(t, method, path, body)
	if username != "" {
		req.Header.Set("X-Forwarded-User", username)
	}
	req.RemoteAddr = remoteAddr
	return req
}

func TestProxyAuth(t *testing.T) {
	mux, data, _, _, cleanup := createTestHandler(t, &Server{
		ProxyAuthUsername: "X-Forwarded-User",
		PrivateRepos:      true,
	})
	defer cleanup()

	// create alice's repository and store a config file
	checkRequest(t, mux.ServeHTTP,
		proxyAuthRequest(t, "POST", "/alice/?create=true", "alice", "", nil),
		[]wantFunc{wantCode(http.StatusOK)})
	checkRequest(t, mux.ServeHTTP,
		proxyAuthRequest(t, "POST", "/alice/config", "alice", "", strings.NewReader(data)),
		[]wantFunc{wantCode(http.StatusOK)})

	for _, test := range []struct {
		name     string
		username string
		want     int
	}{
		{"valid username", "alice", http.StatusOK},
		{"missing username", "", http.StatusUnauthorized},
		{"other user", "bob", http.StatusUnauthorized},
		{"path traversal", "../alice", http.StatusUnauthorized},
		{"whitespace", "alice bob", http.StatusUnauthorized},
		{"newline", "alice\nbob", http.StatusUnauthorized},
		{"separator", "alice;bob", http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkRequest(t, mux.ServeHTTP,
				proxyAuthRequest(t, "GET", "/alice/config", test.username, "", nil),
				[]wantFunc{wantCode(test.want)})
		})
	}
}

func TestProxyAuthTrustedProxies(t *testing.T) {
	mux, data, _, _, cleanup := createTestHandler(t, &Server{
		ProxyAuthUsername: "X-Forwarded-User",
		ProxyHeaders:      true,
		TrustedProxies:    []string{"127.0.0.1", "10.0.0.0/8"},
	})
	defer cleanup()

	// create alice's repository and store a config file via the trusted proxy
	checkRequest(t, mux.ServeHTTP,
		proxyAuthRequest(t, "POST", "/alice/?create=true", "alice", "127.0.0.1:1234", nil),
		[]wantFunc{wantCode(http.StatusOK)})
	checkRequest(t, mux.ServeHTTP,
		proxyAuthRequest(t, "POST", "/alice/config", "alice", "127.0.0.1:1234", strings.NewReader(data)),
		[]wantFunc{wantCode(http.StatusOK)})

	for _, test := range []struct {
		name         string
		username     string
		remoteAddr   string
		forwardedFor string
		want         int
	}{
		{"trusted address", "alice", "127.0.0.1:1234", "", http.StatusOK},
		{"trusted prefix", "alice", "10.1.2.3:1234", "", http.StatusOK},
		{"trusted ipv4-mapped address", "alice", "[::ffff:10.1.2.3]:1234", "", http.StatusOK},
		{"trusted address with forwarded client", "alice", "127.0.0.1:1234", "192.0.2.1", http.StatusOK},
		{"trusted address without username", "", "127.0.0.1:1234", "", http.StatusUnauthorized},
		{"untrusted address", "alice", "192.0.2.1:1234", "", http.StatusUnauthorized},
		{"untrusted ipv6 loopback", "alice", "[::1]:1234", "", http.StatusUnauthorized},
		{"untrusted address with forged X-Forwarded-For", "alice", "192.0.2.1:1234", "127.0.0.1", http.StatusUnauthorized},
		{"no peer address", "alice", "", "", http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := proxyAuthRequest(t, "GET", "/alice/config", test.username, test.remoteAddr, nil)
			if test.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", test.forwardedFor)
			}
			checkRequest(t, mux.ServeHTTP, req, []wantFunc{wantCode(test.want)})
		})
	}
}
