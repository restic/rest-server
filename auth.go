package main

import (
	"errors"
	"net/http"
)

func Authorize(r *http.Request) error {
	username, password, ok := r.BasicAuth()
	if !ok {
		return errors.New("malformed basic auth credentials")
	}

	repo, err := RepositoryName(r.RequestURI)
	if err != nil {
		return err
	}

	if username != "user" || password != "pass" {
		return errors.New("unknown user")
	}

	return nil
}
