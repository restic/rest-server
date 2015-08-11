package handlers

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bchapuis/restic-server/config"
)

func HeadConfig(w http.ResponseWriter, r *http.Request) {
	repo, err := ExtractRepository(r)
	if err != nil {
		http.Error(w, "403 invalid repository", 403)
		return
	}
	file := config.ConfigPath(repo)
	if _, err := os.Stat(file); err != nil {
		http.Error(w, "404 repository not found", 404)
		return
	}
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	repo, err := ExtractRepository(r)
	if err != nil {
		http.Error(w, "403 invalid repository", 403)
		return
	}
	file := config.ConfigPath(repo)
	if _, err := os.Stat(file); err == nil {
		bytes, _ := ioutil.ReadFile(file)
		w.Write(bytes)
		return
	} else {
		http.Error(w, "404 repository not found", 404)
		return
	}
}

func PostConfig(w http.ResponseWriter, r *http.Request) {
	repo, err := ExtractRepository(r)
	if err != nil {
		http.Error(w, "403 invalid repository", 403)
		return
	}
	file := config.ConfigPath(repo)
	if _, err := os.Stat(file); err == nil {
		http.Error(w, "409 repository already initialized", 409)
		return
	} else {
		bytes, _ := ioutil.ReadAll(r.Body)
		ioutil.WriteFile(file, bytes, 0600)
		return
	}
}
