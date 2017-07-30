package restserver

import (
	"path/filepath"
	"testing"
)

func TestJoin(t *testing.T) {
	var tests = []struct {
		base, name string
		result     string
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
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := join(filepath.FromSlash(test.base), test.name)
			if err != nil {
				t.Fatal(err)
			}

			want := filepath.FromSlash(test.result)
			if got != want {
				t.Fatalf("wrong result returned, want %v, got %v", want, got)
			}
		})
	}
}
