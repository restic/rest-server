package main

import (
	//"io/ioutil"
	"log"
	"net/http"
)

func main() {
	context := NewContext("/tmp/restic")

	repo, _ := context.Repository("user")
	repo.Init()

	router := Router{context}
	port := ":8000"
	log.Printf("start server on port %s", port)
	http.ListenAndServe(port, router)
}
