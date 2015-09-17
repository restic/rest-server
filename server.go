package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/restic/restic/backend"
)

const (
	HTTP  = ":8000"
	HTTPS = ":8443"
)

func main() {
	// Parse command-line args
	var path = flag.String("path", "/tmp/restic", "specifies the path of the data directory")
	var tls = flag.Bool("tls", false, "turns on tls support")
	flag.Parse()

	// Create all the necessary subdirectories
	dirs := []string{
		backend.Paths.Data,
		backend.Paths.Snapshots,
		backend.Paths.Index,
		backend.Paths.Locks,
		backend.Paths.Keys,
	}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(*path, d), backend.Modes.Dir)
	}

	router := NewRouter()

	router.Head("/config", func(w http.ResponseWriter, r *http.Request) {
		config := filepath.Join(*path, "config")
		if _, err := os.Stat(config); err != nil {
			http.Error(w, "404 not found", 404)
			return
		}
		w.Write([]byte("200 ok"))
	})

	router.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		config := filepath.Join(*path, "config")
		bytes, err := ioutil.ReadFile(config)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}
		w.Write(bytes)
	})

	router.Post("/config", func(w http.ResponseWriter, r *http.Request) {
		config := filepath.Join(*path, "config")
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "400 bad request", 400)
			return
		}
		errw := ioutil.WriteFile(config, bytes, 0600)
		if errw != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}
		w.Write([]byte("200 ok"))
	})

	router.Get("/:dir/", func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		path := filepath.Join(*path, dir)
		files, err := ioutil.ReadDir(path)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}
		names := make([]string, len(files))
		for i, f := range files {
			names[i] = f.Name()
		}
		data, err := json.Marshal(names)
		if err != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}
		w.Write(data)
	})

	router.Head("/:dir/:name", func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]
		path := filepath.Join(*path, dir, name)
		_, err := os.Stat(path)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}
		w.Write([]byte("200 ok"))
	})

	router.Get("/:type/:name", func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]
		path := filepath.Join(*path, dir, name)
		file, err := os.Open(path)
		if err != nil {
			http.Error(w, "404 not found", 404)
			return
		}
		defer file.Close()
		http.ServeContent(w, r, "", time.Unix(0, 0), file)
	})

	router.Post("/:type/:name", func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]
		path := filepath.Join(*path, dir, name)
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "400 bad request", 400)
			return
		}
		errw := ioutil.WriteFile(path, bytes, 0600)
		if errw != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}
		w.Write([]byte("200 ok"))
	})

	router.Delete("/:type/:name", func(w http.ResponseWriter, r *http.Request) {
		vars := strings.Split(r.RequestURI, "/")
		dir := vars[1]
		name := vars[2]
		path := filepath.Join(*path, dir, name)
		err := os.Remove(path)
		if err != nil {
			http.Error(w, "500 internal server error", 500)
			return
		}
		w.Write([]byte("200 ok"))
	})

	// start the server
	if !*tls {
		log.Printf("start server on port %s", HTTP)
		http.ListenAndServe(HTTP, router)
	} else {
		log.Printf("start server on port %s", HTTPS)
		privateKey := filepath.Join(*path, "private_key")
		publicKey := filepath.Join(*path, "public_key")

		log.Printf("private key: %s", privateKey)
		log.Printf("public key: %s", publicKey)
		http.ListenAndServeTLS(HTTPS, publicKey, privateKey, router)
	}
}
