//go:build !windows
// +build !windows

package repo

import (
	"errors"
	"runtime"
	"syscall"
)

// The ExFAT driver on some versions of macOS can return ENOTTY,
// "inappropriate ioctl for device", for fsync.
//
// https://github.com/restic/restic/issues/4016
// https://github.com/realm/realm-core/issues/5789
func isMacENOTTY(err error) bool {
	return runtime.GOOS == "darwin" && errors.Is(err, syscall.ENOTTY)
}
