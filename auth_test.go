package restserver

import (
	"testing"
)

func TestIsUserPath(t *testing.T) {
	for _, d := range []struct {
		username string
		path     string
		expected bool
	}{
		{"foo", "/", false},
		{"foo", "/foo", true},
		{"foo", "/foo/", true},
		{"foo", "/foo/bar", true},
		{"foo", "/foobar", false},
	} {
		found := isUserPath(d.username, d.path)
		if found != d.expected {
			t.Errorf("expected '%v', found '%v', with %s and %s", d.expected, found, d.username, d.path)
		}
	}
}
