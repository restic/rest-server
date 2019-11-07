package restserver

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
		s.HasError(w, err, 403)
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
		s.HasError(w, errors.New("append only"), 403)
		return
	}

	cfg, err := s.buildPath(getRepoParam(r), "config")
	if s.HasError(w, err, 500) {
		return
	}

	if err := os.Remove(cfg); err != nil {
		if s.conf.Debug {
			s.log.Error(err)
		}
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(404), 404)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		return
	}
}
