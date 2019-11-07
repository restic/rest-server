package tests

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

var loremIpsum = `Lorem ipsum dolor sit amet, consectetur adipisicing elit,
									sed do eiusmod tempor incididunt ut labore et dolore magna
									aliqua. Ut enim ad minim veniam, quis nostrud exercitation
									ullamco laboris nisi ut aliquip ex ea commodo consequat.
									Duis aute irure dolor in reprehenderit in voluptate velit
									esse cillum dolore eu fugiat nulla pariatur. Excepteur sint
									occaecat cupidatat non proident, sunt in culpa qui officia
									deserunt mollit anim id est laborum.`

// TestFsData array
var TestFsData = []struct {
	Path string
	Mode os.FileMode
	Raw  string
}{
	{"dir1", 0755, ""},
	{"dir2", 0755, ""},
	{"dir3", 0750, ""},
	{"file0", 0644, loremIpsum},
	{"dir1/file1", 0644, loremIpsum},
	{"dir2/file2", 0640, loremIpsum},
	{"dir3/file3", 0644, loremIpsum},
	{"dir3/file4", 0644, loremIpsum},
}

// NewEmptyFS returns a tmp file system for testing
func NewEmptyFS() (string, error) {
	return ioutil.TempDir("", "testFsData-")
}

// NewFS returns a tmp file system for testing with fake files
func NewFS() (string, error) {
	fsPath, err := NewEmptyFS()
	if err != nil {
		return "", err
	}

	for _, d := range TestFsData {
		if d.Raw == "" {
			if err := os.MkdirAll(filepath.Join(fsPath, d.Path), d.Mode); err != nil {
				return "", err
			}
		} else {
			if err := ioutil.WriteFile(filepath.Join(fsPath, d.Path), []byte(d.Raw), d.Mode); err != nil {
				return "", err
			}
		}
	}

	return fsPath, nil
}
