package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bchapuis/restic-server/config"
)

func HeadData(w http.ResponseWriter, r *http.Request) {
	repo, err := ExtractRepository(r)
	if err != nil {
		http.Error(w, "403 invalid repository", 403)
		return
	}
	id, err := ExtractID(r)
	if err != nil {
		http.Error(w, "403 invalid ID", 403)
		return
	}
	file := filepath.Join(config.DataPath(repo), id.String())
	if _, err := os.Stat(file); err != nil {
		http.Error(w, "404 repository not found", 404)
		return
	}
}

func GetData(w http.ResponseWriter, r *http.Request) {
	repo, err := ExtractRepository(r)
	if err != nil {
		http.Error(w, "403 invalid repository", 403)
		return
	}
	id, err := ExtractID(r)
	if err != nil {
		http.Error(w, "403 invalid ID", 403)
		return
	}
	file := filepath.Join(config.DataPath(repo), id.String())
	if _, err := os.Stat(file); err != nil {
		http.Error(w, "404 repository not found", 404)
		return
	}
}

func PostData(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "data")
}

func DeleteData(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "data")
}
