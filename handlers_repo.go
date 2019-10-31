package resticserver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// CreateRepo creates repository directories.
func (s *Server) CreateRepo(w http.ResponseWriter, r *http.Request) {
	repoParam, _, _ := getParams(r)

	s.log.Info("DEBUG1")
	repo, err := JoinPaths(s.conf.Path, repoParam)
	if s.HasError(w, err, 500) {
		return
	}

	s.log.Info("DEBUG2")
	if r.URL.Query().Get("create") != "true" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	s.log.Info("DEBUG3")
	log.Printf("Creating repository directories in %s\n", repo)

	if err := os.MkdirAll(repo, 0700); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, d := range ValidTypes {
		if d == "config" {
			continue
		}

		if err := os.MkdirAll(filepath.Join(repo, d), 0700); s.HasError(w, err, 500) {
			return
		}
	}

	for i := 0; i < 256; i++ {
		if err := os.MkdirAll(filepath.Join(repo, "data", fmt.Sprintf("%02x", i)), 0700); s.HasError(w, err, 500) {
			return
		}
	}
}
