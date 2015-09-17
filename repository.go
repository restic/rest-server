package main

import (
	"bytes"
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
		filepath.Join(path, backend.Paths.Data),
		filepath.Join(path, backend.Paths.Snapshots),
		filepath.Join(path, backend.Paths.Index),
		filepath.Join(path, backend.Paths.Locks),
		filepath.Join(path, backend.Paths.Keys),
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

func (r *Repository) ListBlob(t string) ([]string, error) {
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

func (r *Repository) HasBlob(t string, id backend.ID) bool {
	file := filepath.Join(r.path, string(t), id.String())
	if _, err := os.Stat(file); err != nil {
		return false
	}
	return true
}

func (r *Repository) ReadBlob(t string, id backend.ID) (io.ReadSeeker, error) {
	file := filepath.Join(r.path, string(t), id.String())
	blob, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(blob), nil
}

func (r *Repository) WriteBlob(t string, id backend.ID, data []byte) error {
	file := filepath.Join(r.path, string(t), id.String())
	return ioutil.WriteFile(file, data, backend.Modes.File)
}

func (r *Repository) DeleteBlob(t string, id backend.ID) error {
	file := filepath.Join(r.path, string(t), id.String())
	return os.Remove(file)
}
