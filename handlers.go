package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

type Handler func(w http.ResponseWriter, r *http.Request, c *Context)

func HeadConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !repo.HasConfig() {
		http.NotFound(w, r)
		return
	}
}

func GetConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	config, err := repo.ReadConfig()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(config)
}

func PostConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	config, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	errc := repo.WriteConfig(config)
	if errc != nil {
		http.NotFound(w, r)
		return
	}
}

func ListBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if string(bt) == "" {
		http.NotFound(w, r)
		return
	}
	blobs, err := repo.ListBlob(bt)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	json, err := json.Marshal(blobs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(json)
}

func HeadBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if string(bt) == "" {
		http.NotFound(w, r)
		return
	}
	id := BlobID(uri)
	if id.IsNull() {
		http.NotFound(w, r)
		return
	}
	if !repo.HasBlob(bt, id) {
		http.NotFound(w, r)
		return
	}
}

func GetBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if string(bt) == "" {
		http.NotFound(w, r)
		return
	}
	id := BlobID(uri)
	if id.IsNull() {
		http.NotFound(w, r)
		return
	}
	blob, err := repo.ReadBlob(bt, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "", time.Unix(0, 0), blob)
}

func PostBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if string(bt) == "" {
		http.NotFound(w, r)
		return
	}
	id := BlobID(uri)
	if id.IsNull() {
		http.NotFound(w, r)
		return
	}
	blob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	errw := repo.WriteBlob(bt, id, blob)
	if errw != nil {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(200)
}

func DeleteBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, err := RepositoryName(uri)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	repo, err := c.Repository(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if string(bt) == "" {
		http.NotFound(w, r)
		return
	}
	id := BlobID(uri)
	if id.IsNull() {
		http.NotFound(w, r)
		return
	}
	errd := repo.DeleteBlob(bt, id)
	if errd != nil {
		http.NotFound(w, r)
		return
	}
}
