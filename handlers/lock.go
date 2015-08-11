package handlers

import (
	"fmt"
	"net/http"
)

func HeadLock(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "data")
}

func GetLock(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "lock")
}

func PostLock(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "lock")
}

func DeleteLock(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "lock")
}
