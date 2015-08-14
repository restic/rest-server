package main

import ()

// A Context specifies the root directory where all repositories are stored
type Context struct {
	path string
}

func NewContext(path string) Context {
	return Context{path}
}

// Creates the file structure of the Context
func (c *Context) Init() {

}
