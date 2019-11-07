package restserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/restic/rest-server/tests"
)

func TestPathExist(t *testing.T) {
	fsPath, err := tests.NewFS()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(fsPath)

	for _, d := range tests.TestFsData {
		if !PathExist(filepath.Join(fsPath, d.Path)) {
			t.Errorf("expected '%s' to exist", filepath.Join(fsPath, d.Path))
		}
	}
}

func TestPathSize(t *testing.T) {
	fsPath, err := tests.NewFS()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(fsPath)

	size, err := PathSize(fsPath)
	if err != nil {
		t.Error(err)
	}

	if size != 18929 {
		t.Errorf("expected size '%d', found '%d'", 18929, size)
	}
}

// IsValidPath checks whether a path is valid
func TestIsValidPath(t *testing.T) {
	for _, d := range []struct {
		name string
		err  error
	}{
		{"test", nil},
		{"test\x00", errors.New("invalid null character in path")},
		// {"test\\", errors.New("invalid separator character in path")},
	} {
		err := IsValidPath(d.name)
		if err != nil && err.Error() != d.err.Error() {
			t.Errorf("expected error '%v', found '%v'", d.err, err)
		}
	}
}

func TestJoinPaths(t *testing.T) {
	for _, d := range []struct {
		base, name string
		expected   string
	}{
		{"/", "foo/bar", "/foo/bar"},
		{"/srv/server", "foo/bar", "/srv/server/foo/bar"},
		{"/srv/server", "/foo/bar", "/srv/server/foo/bar"},
		{"/srv/server", "foo/../bar", "/srv/server/bar"},
		{"/srv/server", "../bar", "/srv/server/bar"},
		{"/srv/server", "..", "/srv/server"},
		{"/srv/server", "../..", "/srv/server"},
		{"/srv/server", "/repo/data/", "/srv/server/repo/data"},
		{"/srv/server", "/repo/data/../..", "/srv/server"},
		{"/srv/server", "/repo/data/../data/../../..", "/srv/server"},
		{"/srv/server", "/repo/data/../data/../../..", "/srv/server"},
	} {
		t.Run("", func(t *testing.T) {
			found, err := JoinPaths(filepath.FromSlash(d.base), d.name)
			if err != nil {
				t.Fatal(err)
			}

			expected := filepath.FromSlash(d.expected)
			if found != expected {
				t.Errorf("expected result '%v', found '%v'", expected, found)
			}
		})
	}
}
