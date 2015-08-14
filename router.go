package main

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/restic/restic/backend"
)

// Route all the server requests
func Router(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	u := r.RequestURI

	log.Println("%s %s", m, u)

	if Authorize(r) {
		if handler := RestAPI(m, u); handler != nil {
			handler(w, r, nil)
		} else {
			http.Error(w, "not found", 404)
		}
	} else {
		http.Error(w, "unauthorized", 403)
	}
}

// Returns the repository name for a given path
func RepositoryName(u string) (string, error) {
	s := strings.Split(u, "/")
	if len(s) <= 1 {
		return "", errors.New("path does not contain repository name")
	}
	if len(s[1]) < 1 {
		return "", errors.New("repository name should contain at least 1 character")
	}
	match, err := regexp.MatchString("^[a-zA-Z0-9_-]*$", s[1])
	if !match || err != nil {
		return "", errors.New("repository name should not contains special characters")
	}
	return s[1], nil
}

// Returns the backend type for a given path
func BackendType(u string) backend.Type {
	s := strings.Split(u, "/")
	var bt backend.Type
	if len(s) > 2 {
		bt, _ = backend.ParseType(s[2])
	}
	return bt
}

// Returns the blob ID for a given path
func BlobID(u string) backend.ID {
	s := strings.Split(u, "/")
	var id backend.ID
	if len(s) > 3 {
		id, _ = backend.ParseID(s[3])
	}
	return id
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
	if len(s) == 4 && !bt.IsNull() && bt != backend.Config {
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
