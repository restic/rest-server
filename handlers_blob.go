package restserver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/miolini/datacounter"
)

// Blob represents a single blob, its name and its size.
type Blob struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// ListBlobs lists all blobs of a given type in an arbitrary order.
func (s *Server) ListBlobs(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("Accept") {
	case MimeTypeAPIV2:
		s.ListBlobsV2(w, r)
	default:
		s.ListBlobsV1(w, r)
	}
}

// ListBlobsV1 lists all blobs of a given type in an arbitrary order.
func (s *Server) ListBlobsV1(w http.ResponseWriter, r *http.Request) {
	repoParam, typeParam, _ := getParams(r)

	path, err := s.buildPath(repoParam, typeParam)
	if s.HasError(w, err, 500) {
		return
	}

	items, err := ioutil.ReadDir(path)
	if s.HasError(w, err, 404) {
		return
	}

	var names []string
	for _, i := range items {
		if isHashed(typeParam) {
			subpath := filepath.Join(path, i.Name())

			var subitems []os.FileInfo
			subitems, err = ioutil.ReadDir(subpath)
			if s.HasError(w, err, 404) {
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
	if s.HasError(w, err, 500) {
		return
	}

	w.Header().Set("Content-Type", MimeTypeAPIV1)
	_, _ = w.Write(data)
}

// ListBlobsV2 lists all blobs of a given type, together with their sizes, in an arbitrary order.
func (s *Server) ListBlobsV2(w http.ResponseWriter, r *http.Request) {
	repoParam, typeParam, _ := getParams(r)
	path, err := s.buildPath(repoParam, typeParam)
	if s.HasError(w, err, 500) {
		return
	}

	items, err := ioutil.ReadDir(path)
	if s.HasError(w, err, 404) {
		return
	}

	var blobs []Blob
	for _, i := range items {
		if isHashed(typeParam) {
			subpath := filepath.Join(path, i.Name())

			var subitems []os.FileInfo
			subitems, err = ioutil.ReadDir(subpath)
			if s.HasError(w, err, 404) {
				return
			}

			for _, f := range subitems {
				blobs = append(blobs, Blob{Name: f.Name(), Size: f.Size()})
			}
		} else {
			blobs = append(blobs, Blob{Name: i.Name(), Size: i.Size()})
		}
	}

	data, err := json.Marshal(blobs)
	if s.HasError(w, err, 500) {
		return
	}

	w.Header().Set("Content-Type", MimeTypeAPIV2)
	_, _ = w.Write(data)
}

// CheckBlob tests whether a blob exists.
func (s *Server) CheckBlob(w http.ResponseWriter, r *http.Request) {
	repoParam, typeParam, nameParam := getParams(r)
	path, err := s.buildFilePath(repoParam, typeParam, nameParam)
	if s.HasError(w, err, 500) {
		return
	}

	st, err := os.Stat(path)
	if s.HasError(w, err, 404) {
		return
	}

	w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
}

// GetBlob retrieves a blob from the repository.
func (s *Server) GetBlob(w http.ResponseWriter, r *http.Request) {
	repoParam, typeParam, nameParam := getParams(r)
	path, err := s.buildFilePath(repoParam, typeParam, nameParam)
	if s.HasError(w, err, 500) {
		return
	}

	file, err := os.Open(path)
	if s.HasError(w, err, 404) {
		return
	}

	wc := datacounter.NewResponseWriterCounter(w)
	http.ServeContent(wc, r, "", time.Unix(0, 0), file)

	if err = file.Close(); s.HasError(w, err, 500) {
		return
	}

	if s.conf.Prometheus {
		labels := s.getMetricLabels(r)
		metricBlobReadTotal.With(labels).Inc()
		metricBlobReadBytesTotal.With(labels).Add(float64(wc.Count()))
	}
}

// SaveBlob saves a blob to the repository.
func (s *Server) SaveBlob(w http.ResponseWriter, r *http.Request) {
	repoParam, typeParam, nameParam := getParams(r)
	path, err := s.buildFilePath(repoParam, typeParam, nameParam)
	if s.HasError(w, err, 500) {
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
	if os.IsExist(err) {
		http.Error(w, http.StatusText(403), 403)
		return
	}

	if s.HasError(w, err, 500) {
		return
	}

	// ensure this blob does not put us over the repo size limit (if there is one)
	var outFile io.Writer = tf
	if s.conf.MaxRepoSize != 0 {
		var errCode int
		outFile, errCode, err = s.NewWriter(r, tf)
		if err != nil {
			if s.conf.Debug {
				log.Println(err)
			}
			if errCode > 0 {
				http.Error(w, http.StatusText(errCode), errCode)
			}
			return
		}
	}

	written, err := io.Copy(outFile, r.Body)
	if s.HasError(w, err, 400) {
		_ = tf.Close()
		_ = os.Remove(path)
		if s.conf.MaxRepoSize > 0 {
			s.incrementRepoSize(-written)
		}
		return
	}

	if err := tf.Sync(); s.HasError(w, err, 500) {
		_ = tf.Close()
		_ = os.Remove(path)
		if s.conf.MaxRepoSize > 0 {
			s.incrementRepoSize(-written)
		}
		return
	}

	if err := tf.Close(); s.HasError(w, err, 500) {
		_ = os.Remove(path)
		if s.conf.MaxRepoSize > 0 {
			s.incrementRepoSize(-written)
		}
		return
	}

	if s.conf.Prometheus {
		labels := s.getMetricLabels(r)
		metricBlobWriteTotal.With(labels).Inc()
		metricBlobWriteBytesTotal.With(labels).Add(float64(written))
	}
}

// DeleteBlob deletes a blob from the repository.
func (s *Server) DeleteBlob(w http.ResponseWriter, r *http.Request) {
	repoParam, typeParam, nameParam := getParams(r)

	if s.conf.AppendOnly && typeParam != "locks" {
		http.Error(w, http.StatusText(403), 403)
		return
	}

	path, err := s.buildFilePath(repoParam, typeParam, nameParam)
	if s.HasError(w, err, 500) {
		return
	}

	var size int64
	if s.conf.Prometheus || s.conf.MaxRepoSize > 0 {
		stat, err := os.Stat(path)
		if err == nil {
			size = stat.Size()
		}
	}

	if err := os.Remove(path); err != nil {
		if s.conf.Debug {
			log.Print(err)
		}
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(404), 404)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		return
	}

	if s.conf.MaxRepoSize > 0 {
		s.incrementRepoSize(-size)
	}
	if s.conf.Prometheus {
		labels := s.getMetricLabels(r)
		metricBlobDeleteTotal.With(labels).Inc()
		metricBlobDeleteBytesTotal.With(labels).Add(float64(size))
	}
}
