package main

import (
	"errors"
	"net/http"
)

func Authorize(r *http.Request) error {

	htpasswd, err := NewHtpasswdFromFile("/tmp/restic/.htpasswd")
	if err != nil {
		return errors.New("internal server error")
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return errors.New("malformed basic auth credentials")
	}

	if !htpasswd.Validate(username, password) {
		return errors.New("unknown user")
	}

	repo, err := RepositoryName(r.RequestURI)
	if err != nil || repo != username {
		return errors.New("wrong repository")
	}

	return nil
}
