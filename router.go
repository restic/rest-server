package resticserver

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

	router.HandleFunc("/", s.CreateRepo).Methods("POST").Name("global_create_repo")
	router.HandleFunc("/{type}", s.ListBlobs).Methods("GET").Name("global_list_blobs")
	router.HandleFunc("/{type}/{name}", s.CheckBlob).Methods("HEAD").Name("global_check_blob")
	router.HandleFunc("/{type}/{name}", s.DeleteBlob).Methods("DELETE").Name("global_delete_blob")
	router.HandleFunc("/{type}/{name}", s.GetBlob).Methods("GET").Name("global_get_blob")
	router.HandleFunc("/{type}/{name}", s.SaveBlob).Methods("POST").Name("global_save_blob")
	router.HandleFunc("/config", s.CheckConfig).Methods("HEAD").Name("global_check_config")
	router.HandleFunc("/config", s.DeleteConfig).Methods("DELETE").Name("global_delete_config")
	router.HandleFunc("/config", s.GetConfig).Methods("GET").Name("global_get_config")
	router.HandleFunc("/config", s.SaveConfig).Methods("POST").Name("global_save_config")

	router.HandleFunc("/{repo}", s.CreateRepo).Methods("POST").Name("create_repo")
	router.HandleFunc("/{repo}/{type}", s.ListBlobs).Methods("GET").Name("list_blobs")
	router.HandleFunc("/{repo}/{type}/{name}", s.CheckBlob).Methods("HEAD").Name("check_blob")
	router.HandleFunc("/{repo}/{type}/{name}", s.DeleteBlob).Methods("DELETE").Name("delete_blob")
	router.HandleFunc("/{repo}/{type}/{name}", s.GetBlob).Methods("GET").Name("get_blob")
	router.HandleFunc("/{repo}/{type}/{name}", s.SaveBlob).Methods("POST").Name("save_blob")
	router.HandleFunc("/{repo}/config", s.CheckConfig).Methods("HEAD").Name("check_config")
	router.HandleFunc("/{repo}/config", s.DeleteConfig).Methods("DELETE").Name("delete_config")
	router.HandleFunc("/{repo}/config", s.GetConfig).Methods("GET").Name("get_config")
	router.HandleFunc("/{repo}/config", s.SaveConfig).Methods("POST").Name("save_config")

	return router
}

// MethodNotAllowedHandler return a 404 if if method is not allowed
func MethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
}
