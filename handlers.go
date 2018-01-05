package restserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/miolini/datacounter"
	"github.com/prometheus/client_golang/prometheus"
	"goji.io/middleware"
	"goji.io/pat"
)

func isHashed(dir string) bool {
	return dir == "data"
}

func valid(name string) bool {
	// taken from net/http.Dir
	if strings.Contains(name, "\x00") {
		return false
	}

	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return false
	}

	return true
}

var validTypes = []string{"data", "index", "keys", "locks", "snapshots", "config"}

func isValidType(name string) bool {
	for _, tpe := range validTypes {
		if name == tpe {
			return true
		}
	}

	return false
}

// join takes a number of path names, sanitizes them, and returns them joined
// with base for the current operating system to use (dirs separated by
// filepath.Separator). The returned path is always either equal to base or a
// subdir of base.
func join(base string, names ...string) (string, error) {
	clean := make([]string, 0, len(names)+1)
	clean = append(clean, base)

	// taken from net/http.Dir
	for _, name := range names {
		if !valid(name) {
			return "", errors.New("invalid character in path")
		}

		clean = append(clean, filepath.FromSlash(path.Clean("/"+name)))
	}

	return filepath.Join(clean...), nil
}

// getRepo returns the repository location, relative to Config.Path.
func getRepo(r *http.Request) string {
	if strings.HasPrefix(fmt.Sprintf("%s", middleware.Pattern(r.Context())), "/:repo") {
		return pat.Param(r, "repo")
	}

	return "."
}

// getPath returns the path for a file type in the repo.
func getPath(r *http.Request, fileType string) (string, error) {
	if !isValidType(fileType) {
		return "", errors.New("invalid file type")
	}
	return join(Config.Path, getRepo(r), fileType)
}

// getFilePath returns the path for a file in the repo.
func getFilePath(r *http.Request, fileType, name string) (string, error) {
	if !isValidType(fileType) {
		return "", errors.New("invalid file type")
	}

	if isHashed(fileType) {
		if len(name) < 2 {
			return "", errors.New("file name is too short")
		}

		return join(Config.Path, getRepo(r), fileType, name[:2], name)
	}

	return join(Config.Path, getRepo(r), fileType, name)
}

// getUser returns the username from the request, or an empty string if none.
func getUser(r *http.Request) string {
	username, _, ok := r.BasicAuth()
	if !ok {
		return ""
	}
	return username
}

// getMetricLabels returns the prometheus labels from the request.
func getMetricLabels(r *http.Request) prometheus.Labels {
	labels := prometheus.Labels{
		"user": getUser(r),
		"repo": getRepo(r),
		"type": pat.Param(r, "type"),
	}
	return labels
}

// AuthHandler wraps h with a http.HandlerFunc that performs basic authentication against the user/passwords pairs
// stored in f and returns the http.HandlerFunc.
func AuthHandler(f *HtpasswdFile, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if username, password, ok := r.BasicAuth(); !ok || !f.Validate(username, password) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	}
}

// CheckConfig checks whether a configuration exists.
func CheckConfig(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("CheckConfig()")
	}
	cfg, err := getPath(r, "config")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	st, err := os.Stat(cfg)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
}

// GetConfig allows for a config to be retrieved.
func GetConfig(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("GetConfig()")
	}
	cfg, err := getPath(r, "config")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bytes, err := ioutil.ReadFile(cfg)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	_, _ = w.Write(bytes)
}

// SaveConfig allows for a config to be saved.
func SaveConfig(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("SaveConfig()")
	}
	cfg, err := getPath(r, "config")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := ioutil.WriteFile(cfg, bytes, 0600); err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// DeleteConfig removes a config.
func DeleteConfig(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("DeleteConfig()")
	}

	if Config.AppendOnly {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	cfg, err := getPath(r, "config")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := os.Remove(cfg); err != nil {
		if Config.Debug {
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

// ListBlobs lists all blobs of a given type in an arbitrary order.
func ListBlobs(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("ListBlobs()")
	}
	fileType := pat.Param(r, "type")
	path, err := getPath(r, fileType)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	items, err := ioutil.ReadDir(path)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var names []string
	for _, i := range items {
		if isHashed(fileType) {
			subpath := filepath.Join(path, i.Name())
			var subitems []os.FileInfo
			subitems, err = ioutil.ReadDir(subpath)
			if err != nil {
				if Config.Debug {
					log.Print(err)
				}
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			for _, f := range subitems {
				names = append(names, f.Name())
			}
		} else {
			names = append(names, i.Name())
		}
	}

	data, err := json.Marshal(names)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(data)
}

// CheckBlob tests whether a blob exists.
func CheckBlob(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("CheckBlob()")
	}

	path, err := getFilePath(r, pat.Param(r, "type"), pat.Param(r, "name"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	st, err := os.Stat(path)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
}

// GetBlob retrieves a blob from the repository.
func GetBlob(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("GetBlob()")
	}

	path, err := getFilePath(r, pat.Param(r, "type"), pat.Param(r, "name"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	wc := datacounter.NewResponseWriterCounter(w)
	http.ServeContent(wc, r, "", time.Unix(0, 0), file)

	if err = file.Close(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if Config.Prometheus {
		labels := getMetricLabels(r)
		metricBlobReadTotal.With(labels).Inc()
		metricBlobReadBytesTotal.With(labels).Add(float64(wc.Count()))
	}
}

// SaveBlob saves a blob to the repository.
func SaveBlob(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("SaveBlob()")
	}

	path, err := getFilePath(r, pat.Param(r, "type"), pat.Param(r, "name"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tf, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if os.IsNotExist(err) {
		// the error is caused by a missing directory, create it and retry
		mkdirErr := os.MkdirAll(filepath.Dir(path), 0700)
		if mkdirErr != nil {
			log.Print(mkdirErr)
		} else {
			// try again
			tf, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		}
	}

	if err != nil {
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	written, err := io.Copy(tf, r.Body)
	if err != nil {
		_ = tf.Close()
		_ = os.Remove(path)
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := tf.Sync(); err != nil {
		_ = tf.Close()
		_ = os.Remove(path)
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := tf.Close(); err != nil {
		_ = os.Remove(path)
		if Config.Debug {
			log.Print(err)
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if Config.Prometheus {
		labels := getMetricLabels(r)
		metricBlobWriteTotal.With(labels).Inc()
		metricBlobWriteBytesTotal.With(labels).Add(float64(written))
	}
}

// DeleteBlob deletes a blob from the repository.
func DeleteBlob(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("DeleteBlob()")
	}

	if Config.AppendOnly && pat.Param(r, "type") != "locks" {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	path, err := getFilePath(r, pat.Param(r, "type"), pat.Param(r, "name"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var size int64
	if Config.Prometheus {
		stat, err := os.Stat(path)
		if err != nil {
			size = stat.Size()
		}
	}

	if err := os.Remove(path); err != nil {
		if Config.Debug {
			log.Print(err)
		}
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if Config.Prometheus {
		labels := getMetricLabels(r)
		metricBlobDeleteTotal.With(labels).Inc()
		metricBlobDeleteBytesTotal.With(labels).Add(float64(size))
	}
}

// CreateRepo creates repository directories.
func CreateRepo(w http.ResponseWriter, r *http.Request) {
	if Config.Debug {
		log.Println("CreateRepo()")
	}

	repo, err := join(Config.Path, getRepo(r))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("create") != "true" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	log.Printf("Creating repository directories in %s\n", repo)

	if err := os.MkdirAll(repo, 0700); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, d := range validTypes {
		if d == "config" {
			continue
		}

		if err := os.MkdirAll(filepath.Join(repo, d), 0700); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	for i := 0; i < 256; i++ {
		if err := os.MkdirAll(filepath.Join(repo, "data", fmt.Sprintf("%02x", i)), 0700); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}
