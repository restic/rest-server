package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

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

	context := &Context{*path}

	router := NewRouter()
	router.HeadFunc("/config", CheckConfig(context))
	router.GetFunc("/config", GetConfig(context))
	router.PostFunc("/config", SaveConfig(context))
	router.GetFunc("/:dir/", ListBlobs(context))
	router.HeadFunc("/:dir/:name", CheckBlob(context))
	router.GetFunc("/:type/:name", GetBlob(context))
	router.PostFunc("/:type/:name", SaveBlob(context))
	router.DeleteFunc("/:type/:name", DeleteBlob(context))

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
