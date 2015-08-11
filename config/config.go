package config

import (
	"path/filepath"

	"github.com/restic/restic/backend"
)

var root string

func Init(path string) {
	root = path
}

func ConfigPath(repository string) string {
	return filepath.Join(root, repository, string(backend.Config))
}

func DataPath(repository string) string {
	return filepath.Join(root, repository, string(backend.Data))
}

func SnapshotPath(repository string) string {
	return filepath.Join(root, repository, string(backend.Snapshot))
}

func IndexPath(repository string) string {
	return filepath.Join(root, repository, string(backend.Index))
}

func LockPath(repository string) string {
	return filepath.Join(root, repository, string(backend.Lock))
}

func KeyPath(repository string) string {
	return filepath.Join(root, repository, string(backend.Key))
}
