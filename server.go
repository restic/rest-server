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

	router := http.NewServeMux()

	// Check if a configuration exists.
	router.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		uri := r.RequestURI
		method := r.Method
		log.Printf("%s %s", method, uri)

		file := filepath.Join(*path, "config")
		_, err := os.Stat(file)

		// Check if the config exists
		if method == "HEAD" && err == nil {
			return
		}

		// Get the config
		if method == "GET" && err == nil {
			bytes, _ := ioutil.ReadFile(file)
			w.Write(bytes)
			return
		}

		// Save the config
		if method == "POST" && err != nil {
			bytes, _ := ioutil.ReadAll(r.Body)
			ioutil.WriteFile(file, bytes, 0600)
			return
		}

		http.Error(w, "404 not found", 404)
	})

	for _, dir := range dirs {
		router.HandleFunc("/"+dir+"/", func(w http.ResponseWriter, r *http.Request) {
			uri := r.RequestURI
			method := r.Method
			log.Printf("%s %s", method, uri)

			vars := strings.Split(r.RequestURI, "/")
			dir := vars[1]
			name := vars[2]
			path := filepath.Join(*path, dir, name)
			_, err := os.Stat(path)

			// List the blobs of a given dir.
			if method == "GET" && name == "" && err == nil {
				files, _ := ioutil.ReadDir(path)
				names := make([]string, len(files))
				for i, f := range files {
					names[i] = f.Name()
				}
				data, _ := json.Marshal(names)
				w.Write(data)
				return
			}

			// Check if the blob esists
			if method == "HEAD" && name != "" && err == nil {
				return
			}

			// Get a blob of a given dir.
			if method == "GET" && name != "" && err == nil {
				file, _ := os.Open(path)
				defer file.Close()
				http.ServeContent(w, r, "", time.Unix(0, 0), file)
				return
			}

			// Save a blob
			if method == "POST" && name != "" && err != nil {
				bytes, _ := ioutil.ReadAll(r.Body)
				ioutil.WriteFile(path, bytes, 0600)
				return
			}

			// Delete a blob
			if method == "DELETE" && name != "" && err == nil {
				os.Remove(path)
				return
			}

			http.Error(w, "404 not found", 404)
		})
	}

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
