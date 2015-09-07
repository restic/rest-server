package main

import (
	"os"
	"path/filepath"

	"github.com/restic/restic/backend"
)

// A Context specifies the root directory where all repositories are stored
type Context struct {
	path string
}

func NewContext(path string) Context {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); err != nil {
		os.MkdirAll(path, backend.Modes.Dir)
	}
	return Context{path}
}

func (c *Context) Repository(name string) (*Repository, error) {
	name, err := ParseRepositoryName(name)
	if err != nil {
		return nil, err
	}

	repo, err := NewRepository(filepath.Join(c.path, name))
	if err != nil {
		return nil, err
	}

	return repo, nil
}
