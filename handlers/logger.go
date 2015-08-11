package handlers

import (
	"log"
	"net/http"
)

func RequestLogger(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v %v", r.Method, r.URL.String())
}
