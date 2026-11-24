package repo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/restic/rest-server/quota"
)

func TestIsInsufficientStorage(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{"quota", fmt.Errorf("write: %w", quota.ErrMaxSizeExceeded), true},
		{"ENOSPC", &os.PathError{Op: "write", Path: "config", Err: syscall.ENOSPC}, true},
		{"EDQUOT", &os.PathError{Op: "write", Path: "config", Err: syscall.EDQUOT}, true},
		{"wrapped ENOSPC", fmt.Errorf("copy: %w", &os.PathError{Op: "write", Path: "config", Err: syscall.ENOSPC}), true},
		{"other path error", &os.PathError{Op: "write", Path: "config", Err: syscall.EIO}, false},
		{"unexpected EOF", io.ErrUnexpectedEOF, false},
		{"other", errors.New("boom"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isInsufficientStorage(test.err); got != test.want {
				t.Errorf("isInsufficientStorage(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
