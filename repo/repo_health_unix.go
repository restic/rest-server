//go:build !windows
// +build !windows

package repo

import (
	"errors"
	"io/fs"
	"syscall"

	"golang.org/x/sys/unix"
)

func isWritable(path string) (bool, error) {
	err := unix.Access(path, unix.W_OK)
	var err2 syscall.Errno
	if errors.As(err, &err2) && err2.Is(fs.ErrPermission) {
		return false, nil
	}
	return err == nil, err
}

func getFreeSpace(path string) (uint64, error) {
	var stat unix.Statfs_t

	err := unix.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}

	// Available blocks * size per block = available space in bytes
	return stat.Bavail * uint64(stat.Bsize), nil
}
