package main

import (
	"errors"
	"net/http"
	"path/filepath"
)

func Authorize(r *http.Request, c *Context) error {

	file := filepath.Join(c.path, ".htpasswd")
	htpasswd, err := NewHtpasswdFromFile(file)
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
