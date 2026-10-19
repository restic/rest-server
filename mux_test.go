package restserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestCheckAuthLogsRejectedCredentials(t *testing.T) {
	// user "restic" with password "test"
	server := &Server{HtpasswdPath: writeHtpasswd(t, "restic:$2y$05$z/OEmNQamd6m6LSegUErh.r/Owk9Xwmc5lxDheIuHY2Z7XiS6FtJm\n")}
	if _, err := NewHandler(server); err != nil {
		t.Fatal(err)
	}
	var lines []string
	server.authFailures.printf = func(format string, v ...any) {
		lines = append(lines, fmt.Sprintf(format, v...))
	}

	tests := []struct {
		name          string
		remoteAddr    string
		noCredentials bool
		user          string
		password      string
		wantOk        bool
		wantLog       string
	}{
		{
			name:       "wrong password",
			remoteAddr: "192.0.2.1:51234",
			user:       "restic",
			password:   "wrong",
			wantLog:    `Rejected authentication for user "restic" from 192.0.2.1`,
		},
		{
			name:       "unknown user",
			remoteAddr: "[2001:db8::1]:51234",
			user:       "nobody",
			password:   "test",
			wantLog:    `Rejected authentication for user "nobody" from 2001:db8::1`,
		},
		{
			name:       "username is escaped",
			remoteAddr: "192.0.2.2:51234",
			user:       "no\nbody",
			password:   "test",
			wantLog:    `Rejected authentication for user "no\nbody" from 192.0.2.2`,
		},
		{
			name:       "address from proxy headers",
			remoteAddr: "192.0.2.3",
			user:       "restic",
			password:   "wrong",
			wantLog:    `Rejected authentication for user "restic" from 192.0.2.3`,
		},
		{
			name:          "no credentials",
			remoteAddr:    "192.0.2.4:51234",
			noCredentials: true,
		},
		{
			name:       "correct password",
			remoteAddr: "192.0.2.5:51234",
			user:       "restic",
			password:   "test",
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines = nil
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if !tt.noCredentials {
				req.SetBasicAuth(tt.user, tt.password)
			}

			_, ok := server.checkAuth(req)
			if ok != tt.wantOk {
				t.Errorf("checkAuth ok = %v, want %v", ok, tt.wantOk)
			}

			var want []string
			if tt.wantLog != "" {
				want = []string{tt.wantLog}
			}
			if fmt.Sprint(lines) != fmt.Sprint(want) {
				t.Errorf("log messages = %q, want %q", lines, want)
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
