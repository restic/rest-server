// +build !windows

package repo

import "golang.org/x/sys/unix"

func isWritable(path string) (bool, error) {
	err := unix.Access(path, unix.W_OK)
	return err == nil, err
}
