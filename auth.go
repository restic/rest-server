package restserver

import (
	"net/http"
	"strings"
)

// AuthMiddleware performs basic authentication against the user/passwords pairs
// stored in the htpasswd file
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || !s.auth.Validate(username, password) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if s.conf.PrivateRepos && !isUserPath(username, r.URL.Path) && r.URL.Path != "/metrics" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isUserPath checks if a request path is accessible by the user
func isUserPath(username, path string) bool {
	prefix := "/" + username
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	return len(path) == len(prefix) || path[len(prefix)] == '/'
}
