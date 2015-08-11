package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/restic/restic/backend"
)

func ExtractUser(r *http.Request) (string, string, error) {
	return "username", "password", nil
}

func ExtractRepository(r *http.Request) (string, error) {
	return "repository", nil
}

func ExtractID(r *http.Request) (backend.ID, error) {
	path := strings.Split(r.URL.String(), "/")
	if len(path) != 3 {
		return backend.ID{}, errors.New("invalid request path")
	}
	return backend.ParseID(path[2])
}
