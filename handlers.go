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
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
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
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	config, errrc := repo.ReadConfig()
	if errrc != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(config)
}

func PostConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	config, errc := ioutil.ReadAll(r.Body)
	if errc != nil {
		http.NotFound(w, r)
		return
	}
	errwc := repo.WriteConfig(config)
	if errwc != nil {
		http.NotFound(w, r)
		return
	}
}

func ListBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if bt.IsNull() {
		http.NotFound(w, r)
		return
	}
	blobs, errb := repo.ListBlob(bt)
	if errb != nil {
		http.NotFound(w, r)
		return
	}
	json, errj := json.Marshal(blobs)
	if errj != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(json)
}

func HeadBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if bt.IsNull() {
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
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if bt.IsNull() {
		http.NotFound(w, r)
		return
	}
	id := BlobID(uri)
	if id.IsNull() {
		http.NotFound(w, r)
		return
	}
	blob, errb := repo.ReadBlob(bt, id)
	if errb != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "", time.Unix(0, 0), blob)
}

func PostBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if bt.IsNull() {
		http.NotFound(w, r)
		return
	}
	id := BlobID(uri)
	if id.IsNull() {
		http.NotFound(w, r)
		return
	}
	blob, errb := ioutil.ReadAll(r.Body)
	if errb != nil {
		http.NotFound(w, r)
		return
	}
	errwb := repo.WriteBlob(bt, id, blob)
	if errwb != nil {
		http.NotFound(w, r)
		return
	}
}

func DeleteBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	uri := r.RequestURI
	name, errrn := RepositoryName(uri)
	if errrn != nil {
		http.NotFound(w, r)
		return
	}
	repo, errr := c.Repository(name)
	if errr != nil {
		http.NotFound(w, r)
		return
	}
	bt := BackendType(uri)
	if bt.IsNull() {
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
