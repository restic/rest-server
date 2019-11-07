package restserver

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter returns the main http router
func (s *Server) NewRouter() *mux.Router {
	router := mux.NewRouter()

	router.MethodNotAllowedHandler = MethodNotAllowedHandler()

	router.StrictSlash(true)

	if s.conf.Prometheus {
		router.Handle("/metrics", promhttp.Handler())
	}

	// Routes order matters !
	router.HandleFunc("/", s.CreateRepo).Methods("POST")
	router.HandleFunc("/config", s.CheckConfig).Methods("HEAD")
	router.HandleFunc("/config", s.DeleteConfig).Methods("DELETE")
	router.HandleFunc("/config", s.GetConfig).Methods("GET")
	router.HandleFunc("/config", s.SaveConfig).Methods("POST")
	router.HandleFunc("/{type}", s.ListBlobs).Methods("GET")
	router.HandleFunc("/{type}/{name}", s.CheckBlob).Methods("HEAD")
	router.HandleFunc("/{type}/{name}", s.DeleteBlob).Methods("DELETE")
	router.HandleFunc("/{type}/{name}", s.GetBlob).Methods("GET")
	router.HandleFunc("/{type}/{name}", s.SaveBlob).Methods("POST")

	router.HandleFunc("/{repo}", s.CreateRepo).Methods("POST")
	router.HandleFunc("/{repo}/config", s.CheckConfig).Methods("HEAD")
	router.HandleFunc("/{repo}/config", s.DeleteConfig).Methods("DELETE")
	router.HandleFunc("/{repo}/config", s.GetConfig).Methods("GET")
	router.HandleFunc("/{repo}/config", s.SaveConfig).Methods("POST")
	router.HandleFunc("/{repo}/{type}", s.ListBlobs).Methods("GET")
	router.HandleFunc("/{repo}/{type}/{name}", s.CheckBlob).Methods("HEAD")
	router.HandleFunc("/{repo}/{type}/{name}", s.DeleteBlob).Methods("DELETE")
	router.HandleFunc("/{repo}/{type}/{name}", s.GetBlob).Methods("GET")
	router.HandleFunc("/{repo}/{type}/{name}", s.SaveBlob).Methods("POST")

	return router
}

// MethodNotAllowedHandler return a 404 if if method is not allowed
func MethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
}
