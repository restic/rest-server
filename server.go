package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
)

const (
	HTTP  = ":8000"
	HTTPS = ":8443"
)

func main() {
	var path = flag.String("path", "/tmp/restic", "specifies the path of the data directory")
	var tls = flag.Bool("tls", false, "turns on tls support")
	flag.Parse()

	context := NewContext(*path)
	router := Router{context}
	if !*tls {
		log.Printf("start server on port %s", HTTP)
		http.ListenAndServe(HTTP, router)
	} else {
		log.Printf("start server on port %s", HTTPS)
		privateKey := filepath.Join(*path, "private_key")
		publicKey := filepath.Join(*path, "public_key")
		http.ListenAndServeTLS(HTTPS, privateKey, publicKey, router)
	}
}
