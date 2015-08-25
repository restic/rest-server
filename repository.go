package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/restic/restic/backend"
)

// A Repository is the place where backups are stored
type Repository struct {
	path string
}

// Creates the file structure of the Repository
func (r *Repository) Init() error {
	dirs := []string{
		r.path,
		filepath.Join(r.path, string(backend.Data)),
		filepath.Join(r.path, string(backend.Snapshot)),
		filepath.Join(r.path, string(backend.Index)),
		filepath.Join(r.path, string(backend.Lock)),
		filepath.Join(r.path, string(backend.Key)),
	}
	for _, d := range dirs {
		if _, errs := os.Stat(d); errs != nil {
			errmk := os.MkdirAll(d, backend.Modes.Dir)
			if errmk != nil {
				return errmk
			}
		}
	}
	return nil
}

func (r *Repository) HasConfig() bool {
	file := filepath.Join(r.path, string(backend.Config))
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return true
}

func (r *Repository) ReadConfig() ([]byte, error) {
	file := filepath.Join(r.path, string(backend.Config))
	return ioutil.ReadFile(file)
}

func (r *Repository) WriteConfig(data []byte) error {
	file := filepath.Join(r.path, string(backend.Config))
	return ioutil.WriteFile(file, data, backend.Modes.File)
}

func (r *Repository) ListBlob(t backend.Type) ([]string, error) {
	var blobs []string
	dir := filepath.Join(r.path, string(t))
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return blobs, err
	}
	blobs = make([]string, len(files))
	for i, f := range files {
		blobs[i] = f.Name()
	}
	return blobs, nil
}

func (r *Repository) HasBlob(bt backend.Type, id backend.ID) bool {
	file := filepath.Join(r.path, string(bt), id.String())
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return true
}

func (r *Repository) ReadBlob(bt backend.Type, id backend.ID) (io.ReadSeeker, error) {
	file := filepath.Join(r.path, string(bt), id.String())
	f, err := os.Open(file)
	if err != nil {
		return f, err
	}
	return f, nil
}

func (r *Repository) WriteBlob(bt backend.Type, id backend.ID, data []byte) error {
	file := filepath.Join(r.path, string(bt), id.String())
	return ioutil.WriteFile(file, data, backend.Modes.File)
}

func (r *Repository) DeleteBlob(bt backend.Type, id backend.ID) error {
	file := filepath.Join(r.path, string(bt), id.String())
	return os.Remove(file)
}
