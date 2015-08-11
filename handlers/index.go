package handlers

import (
	"fmt"
	"net/http"
)

func HeadIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "data")
}

func GetIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "index")
}

func PostIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "index")
}

func DeleteIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "index")
}
