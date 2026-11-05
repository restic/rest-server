package restserver

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

// rawResponse is the part of a response that the tests look at.
type rawResponse struct {
	status int
	close  bool // the server announced that it closes the connection
	body   string
}

// sendRaw sends a raw HTTP/1.1 request on conn and reads the response. It
// fails if no response arrives within a few seconds.
func sendRaw(t *testing.T, conn net.Conn, br *bufio.Reader, request string) rawResponse {
	t.Helper()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(conn, request); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("no response from server: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return rawResponse{status: resp.StatusCode, close: resp.Close, body: string(body)}
}

// TestCloseOnUnreadBody checks that a client cannot stall the server by not
// sending the body of a request that the server answers without reading it.
func TestCloseOnUnreadBody(t *testing.T) {
	mux, _, _, _, cleanup := createTestHandler(t, &Server{
		ProxyAuthUsername: "X-Remote-User",
		Prometheus:        true,
		PrometheusNoAuth:  true,
	})
	defer cleanup()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tests := []struct {
		name       string
		request    string
		wantStatus int
	}{
		{
			name:       "unauthenticated upload",
			request:    "POST /config HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1000\r\n\r\npartial",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unauthenticated chunked upload",
			request:    "POST /config HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n7\r\npartial\r\n",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "health check with body",
			request:    "POST /healthz HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1000\r\n\r\npartial",
			wantStatus: http.StatusNoContent,
		},
		{
			// the response is large enough to be sent while the handler
			// is still running
			name:       "metrics with body",
			request:    "POST /metrics HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1000\r\n\r\npartial",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := net.Dial("tcp", srv.Listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = conn.Close() }()
			br := bufio.NewReader(conn)

			// the body is incomplete and the client stalls
			resp := sendRaw(t, conn, br, tt.request)
			if resp.status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.status, tt.wantStatus)
			}
			if !resp.close {
				t.Error("response does not announce that the connection is closed")
			}

			// the server must close the connection instead of waiting for
			// the rest of the body
			_, err = br.ReadByte()
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				t.Fatal("server did not close the connection")
			}
			if err == nil {
				t.Fatal("unexpected data after the response")
			}
		})
	}
}

// TestKeepAliveAfterUpload checks that a connection stays open once the
// handler has read the whole request body.
func TestKeepAliveAfterUpload(t *testing.T) {
	mux, _, _, _, cleanup := createTestHandler(t, &Server{ProxyAuthUsername: "X-Remote-User"})
	defer cleanup()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	br := bufio.NewReader(conn)

	resp := sendRaw(t, conn, br, "POST /config HTTP/1.1\r\nHost: localhost\r\nX-Remote-User: user\r\nContent-Length: 4\r\n\r\ntest")
	if resp.status != http.StatusOK {
		t.Fatalf("upload: status = %d, want %d", resp.status, http.StatusOK)
	}
	if resp.close {
		t.Fatal("upload: server closes the connection")
	}

	// a second request on the same connection must still work
	resp = sendRaw(t, conn, br, "GET /config HTTP/1.1\r\nHost: localhost\r\nX-Remote-User: user\r\n\r\n")
	if resp.status != http.StatusOK || resp.body != "test" {
		t.Fatalf("download: status = %d, body = %q, want %d, %q", resp.status, resp.body, http.StatusOK, "test")
	}
}
