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

func NewRepository(path string) (*Repository, error) {
	dirs := []string{
		path,
		filepath.Join(path, string(backend.Data)),
		filepath.Join(path, string(backend.Snapshot)),
		filepath.Join(path, string(backend.Index)),
		filepath.Join(path, string(backend.Lock)),
		filepath.Join(path, string(backend.Key)),
	}
	for _, d := range dirs {
		_, err := os.Stat(d)
		if err != nil {
			err := os.MkdirAll(d, backend.Modes.Dir)
			if err != nil {
				return nil, err
			}
		}
	}
	return &Repository{path}, nil
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
	defer f.Close()
	return f, err
}

func (r *Repository) WriteBlob(bt backend.Type, id backend.ID, data []byte) error {
	file := filepath.Join(r.path, string(bt), id.String())
	return ioutil.WriteFile(file, data, backend.Modes.File)
}

func (r *Repository) DeleteBlob(bt backend.Type, id backend.ID) error {
	file := filepath.Join(r.path, string(bt), id.String())
	return os.Remove(file)
}
