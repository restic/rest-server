package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Context contains repository metadata.
type Context struct {
	path string
}

// AuthHandler wraps h with a http.HandlerFunc that performs basic authentication against the user/passwords pairs
// stored in f and returns the http.HandlerFunc.
func AuthHandler(f *HtpasswdFile, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if username, password, ok := r.BasicAuth(); !ok || !f.Validate(username, password) {
			http.Error(w, "401 unauthorized", 401)
			return
		}

		h.ServeHTTP(w, r)
	}
}

// CheckConfig returns a http.HandlerFunc that checks whether a configuration exists.
func CheckConfig(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := filepath.Join(c.path, "config")
		st, err := os.Stat(config)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}

		w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
	}
}

// GetConfig returns a http.HandlerFunc that allows for a config to be retrieved.
func GetConfig(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := filepath.Join(c.path, "config")
		bytes, err := ioutil.ReadFile(config)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}

		w.Write(bytes)
	}
}

// SaveConfig returns a http.HandlerFunc that allows for a config to be saved.
func SaveConfig(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := filepath.Join(c.path, "config")
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "400 bad request", 400)
			return
		}
		if err := ioutil.WriteFile(config, bytes, 0600); err != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}

		w.Write([]byte("200 ok"))
	}
}

// ListBlobs returns a http.HandlerFunc that lists all blobs of a given type in an arbitrary order.
func ListBlobs(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		path := filepath.Join(c.path, dir)

		items, err := ioutil.ReadDir(path)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}

		var names []string
		for _, i := range items {
			if dir == "data" {
				subpath := filepath.Join(path, i.Name())
				subitems, err := ioutil.ReadDir(subpath)
				if err != nil {
					http.Error(w, "404 not found", 404)
					return
				}
				for _, f := range subitems {
					names = append(names, f.Name())
				}
			} else {
				names = append(names, i.Name())
			}
		}

		data, err := json.Marshal(names)
		if err != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}

		w.Write(data)
	}
}

// CheckBlob returns a http.HandlerFunc that tests whether a blob exists and returns 200, if it does, or 404 otherwise.
func CheckBlob(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]

		if dir == "data" {
			name = filepath.Join(name[:2], name)
		}
		path := filepath.Join(c.path, dir, name)

		st, err := os.Stat(path)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}

		w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
	}
}

// GetBlob returns a http.HandlerFunc that retrieves a blob from the repository.
func GetBlob(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]

		if dir == "data" {
			name = filepath.Join(name[:2], name)
		}
		path := filepath.Join(c.path, dir, name)

		file, err := os.Open(path)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}

		http.ServeContent(w, r, "", time.Unix(0, 0), file)
		file.Close()
	}
}

// SaveBlob returns a http.HandlerFunc that saves a blob to the repository.
func SaveBlob(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]

		tmp := filepath.Join(c.path, "tmp", name)

		tf, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}

		if _, err := io.Copy(tf, r.Body); err != nil {
			tf.Close()
			os.Remove(tmp)
			http.Error(w, "400 bad request", 400)
			return
		}
		if err := tf.Sync(); err != nil {
			tf.Close()
			os.Remove(tmp)
			http.Error(w, "500 internal server error", 500)
			return
		}
		if err := tf.Close(); err != nil {
			os.Remove(tmp)
			http.Error(w, "500 internal server error", 500)
			return
		}

		if dir == "data" {
			name = filepath.Join(name[:2], name)
		}
		path := filepath.Join(c.path, dir, name)

		if err := os.Rename(tmp, path); err != nil {
			os.Remove(tmp)
			os.Remove(path)
			http.Error(w, "500 internal server error", 500)
			return
		}

		w.Write([]byte("200 ok"))
	}
}

// DeleteBlob returns a http.HandlerFunc that deletes a blob from the repository.
func DeleteBlob(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]

		if dir == "data" {
			name = filepath.Join(name[:2], name)
		}
		path := filepath.Join(c.path, dir, name)

		if err := os.Remove(path); err != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}

		w.Write([]byte("200 ok"))
	}
}
