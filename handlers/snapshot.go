package handlers

import (
	"fmt"
	"net/http"
)

func HeadSnapshot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "data")
}

func GetSnapshot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "snapshot")
}

func PostSnapshot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "snapshot")
}

func DeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "snapshot")
}
