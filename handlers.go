package restserver

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

// MimeTypeAPI are mime types used for versionning
const (
	MimeTypeAPIV1 = "application/vnd.x.restic.rest.v1"
	MimeTypeAPIV2 = "application/vnd.x.restic.rest.v2"
)

// HasError checks whether there is an error, and process the error
func (s *Server) HasError(w http.ResponseWriter, err error, failStatus int) bool {
	if err != nil {
		if s.conf.Debug {
			s.log.Error(err)
		}
		http.Error(w, http.StatusText(failStatus), failStatus)
		return true
	}
	return false
}

func getParams(r *http.Request) (string, string, string) {
	params := mux.Vars(r)

	repoParam, ok := params["repo"]
	if !ok || repoParam == "" {
		repoParam = "."
	}

	typeParam, ok := params["type"]
	if !ok {
		typeParam = ""
	}

	nameParam, ok := params["name"]
	if !ok {
		nameParam = ""
	}

	return repoParam, typeParam, nameParam
}

func getRepoParam(r *http.Request) string {
	repoParam, ok := mux.Vars(r)["repo"]
	if !ok {
		return ""
	}

	// If repoParam is not a valid type, it has to be a repo else, use "."
	if err := IsValidType(repoParam); err != nil {
		fmt.Printf("DEBUG: returning %s", repoParam)
		return repoParam
	}

	fmt.Printf("DEBUG: returning %s", ".")
	return "."
}

func getTypeParam(r *http.Request) string {
	typeParam, ok := mux.Vars(r)["type"]
	if !ok {
		return ""
	}
	return typeParam
}

func getNameParam(r *http.Request) string {
	nameParam, ok := mux.Vars(r)["name"]
	if !ok {
		return ""
	}
	return nameParam
}

func getUser(r *http.Request) string {
	username, _, ok := r.BasicAuth()
	if !ok {
		return ""
	}
	return username
}

// getMetricLabels returns the prometheus labels from the request.
func (s *Server) getMetricLabels(r *http.Request) prometheus.Labels {
	rep, typ, _ := getParams(r)
	labels := prometheus.Labels{
		"user": getUser(r),
		"repo": rep,
		"type": typ,
	}
	return labels
}
