package restserver

import (
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
