package main

import (
	"errors"
	"regexp"
	"strings"

	"github.com/restic/restic/backend"
)

// Returns the repository name for a given path
func RepositoryName(u string) (string, error) {
	s := strings.Split(u, "/")
	if len(s) <= 1 {
		return "", errors.New("path does not contain repository name")
	}
	return ParseRepositoryName(s[1])
}

func ParseRepositoryName(n string) (string, error) {
	if len(n) < 1 {
		return "", errors.New("repository name should contain at least 1 character")
	}
	match, err := regexp.MatchString("^[a-zA-Z0-9_-]*$", n)
	if !match || err != nil {
		return "", errors.New("repository name should not contains special characters")
	}
	return n, nil
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
