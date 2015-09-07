package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/restic/restic/backend"
)

type Router struct {
	Context
}

func NewRouter(context Context) *Router {
	return &Router{context}
}

func (router Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	u := r.RequestURI

	log.Printf("%s %s", m, u)

	if err := Authorize(r, &router.Context); err == nil {
		if handler := RestAPI(m, u); handler != nil {
			handler(w, r, &router.Context)
		} else {
			http.Error(w, "not found", 404)
		}
	} else {
		http.Error(w, err.Error(), 403)
	}
}

// The Rest API returns a Handler when a match occur or nil.
func RestAPI(m string, u string) Handler {
	s := strings.Split(u, "/")

	// Check for valid repository name
	_, err := RepositoryName(u)
	if err != nil {
		return nil
	}

	// Route config requests
	bt := BackendType(u)
	if len(s) == 3 && bt == backend.Config {
		switch m {
		case "HEAD":
			return HeadConfig
		case "GET":
			return GetConfig
		case "POST":
			return PostConfig
		}
	}

	// Route blob requests
	id := BlobID(u)
	if len(s) == 4 && string(bt) != "" && bt != backend.Config {
		if s[3] == "" && m == "GET" {
			return ListBlob
		} else if !id.IsNull() {
			switch m {
			case "HEAD":
				return HeadBlob
			case "GET":
				return GetBlob
			case "POST":
				return PostBlob
			case "DELETE":
				return DeleteBlob
			}
		}
	}

	return nil
}
