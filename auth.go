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

	if username != "user" || password != "pass" {
		return errors.New("unknown user")
	}

	return nil
}
