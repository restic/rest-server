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
	return Context{filepath.Clean(path)}
}

// Creates the file structure of the Context
func (c *Context) Init() error {
	if _, err := os.Stat(c.path); err != nil {
		return os.MkdirAll(c.path, backend.Modes.Dir)
	}
	return nil
}

func (c *Context) Repository(name string) (Repository, error) {
	name, err := ParseRepositoryName(name)
	return Repository{filepath.Join(c.path, name)}, err
}
