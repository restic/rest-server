package resticserver

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// CheckConfig checks whether a configuration exists.
func (s *Server) CheckConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.buildPath(getRepoParam(r), "config")
	if s.HasError(w, err, 500) {
		return
	}

	st, err := os.Stat(cfg)
	if s.HasError(w, err, 404) {
		return
	}

	w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
}

// GetConfig allows for a config to be retrieved.
func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.buildPath(getRepoParam(r), "config")
	if s.HasError(w, err, 500) {
		return
	}

	bytes, err := ioutil.ReadFile(cfg)
	if s.HasError(w, err, 404) {
		return
	}

	_, _ = w.Write(bytes)
}

// SaveConfig allows for a config to be saved.
func (s *Server) SaveConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.buildPath(getRepoParam(r), "config")
	if s.HasError(w, err, 500) {
		return
	}

	f, err := os.OpenFile(cfg, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil && os.IsExist(err) {
		if s.conf.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	_, err = io.Copy(f, r.Body)
	if s.HasError(w, err, 500) {
		return
	}

	err = f.Close()
	if s.HasError(w, err, 500) {
		return
	}

	_ = r.Body.Close()
}

// DeleteConfig removes a config.
func (s *Server) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	if s.conf.AppendOnly {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	cfg, err := s.buildPath(getRepoParam(r), "config")
	if s.HasError(w, err, 500) {
		return
	}

	if err := os.Remove(cfg); err != nil {
		if s.conf.Debug {
			log.Print(err)
		}
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
}
