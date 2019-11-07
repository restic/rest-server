package restserver

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// PathExist check wether a file/dir exists
func PathExist(path string) bool {
	path = filepath.Clean(path)
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// PathSize counts the size of the contents of path.
func PathSize(path string) (int64, error) {
	if path == "" {
		path = "."
	}
	var size int64
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}

// IsValidPath checks whether a path is valid
func IsValidPath(name string) error {
	if strings.Contains(name, "\x00") {
		return errors.New("invalid null character in path")
	}

	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return errors.New("invalid separator character in path")
	}

	return nil
}

// JoinPaths takes a number of path names, sanitizes them, and returns them joined
// with base for the current operating system to use (dirs separated by
// filepath.Separator). The returned path is always either equal to base or a
// subdir of base.
func JoinPaths(base string, names ...string) (string, error) {
	clean := make([]string, 0, len(names)+1)
	clean = append(clean, base)

	for _, name := range names {
		if err := IsValidPath(name); err != nil {
			return "", err
		}

		clean = append(clean, filepath.FromSlash(path.Clean("/"+name)))
	}

	return filepath.Join(clean...), nil
}

// buildPath returns the path for a file type in the repo
func (s *Server) buildPath(repoParam, typeParam string) (string, error) {
	if err := IsValidType(typeParam); err != nil {
		return "", err
	}

	return JoinPaths(s.conf.Path, repoParam, typeParam)
}

// buildFilePath returns the path for a file in the repo.
func (s *Server) buildFilePath(repoParam, typeParam, nameParam string) (string, error) {
	if err := IsValidType(typeParam); err != nil {
		return "", err
	}

	if isHashed(typeParam) {
		if len(nameParam) < 2 {
			return "", errors.New("file name is too short")
		}
		return JoinPaths(s.conf.Path, repoParam, typeParam, nameParam[:2], nameParam)
	}

	return JoinPaths(s.conf.Path, repoParam, typeParam, nameParam)
}
