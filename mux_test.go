package restserver

import (
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
