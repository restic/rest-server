package handlers

import (
	"fmt"
	"net/http"
)

func HeadKey(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "data")
}

func GetKey(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "key")
}

func PostKey(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "key")
}

func DeleteKey(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "key")
}
