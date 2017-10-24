package restserver

/*
Original version copied from: github.com/bitly/oauth2_proxy

MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/csv"
	"log"
	"os"
	"sync"
	"time"
)

const CheckInterval = 30 * time.Second

// Lookup passwords in a htpasswd file.  The entries must have been created with -s for SHA encryption.

// HtpasswdFile is a map for usernames to passwords.
type HtpasswdFile struct {
	mutex    sync.Mutex
	path     string
	stat     os.FileInfo
	throttle chan struct{}
	Users    map[string]string
}

// NewHtpasswdFromFile reads the users and passwords from a htpasswd file and returns them.  If an error is encountered,
// it is returned, together with a nil-Pointer for the HtpasswdFile.
func NewHtpasswdFromFile(path string) (*HtpasswdFile, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	h := &HtpasswdFile{
		mutex:    sync.Mutex{},
		path:     path,
		stat:     stat,
		throttle: make(chan struct{}),
	}

	if err := h.Reload(); err != nil {
		return nil, err
	}

	// Start a goroutine that limits reload checks to once per CheckInterval
	go h.throttleTimer()

	return h, nil
}

// throttleTimer sends at most one message per CheckInterval to throttle file change checks.
func (h *HtpasswdFile) throttleTimer() {
	var check struct{}
	for {
		time.Sleep(CheckInterval)
		h.throttle <- check
	}
}

// Reload reloads the htpasswd file. If the reload fails, the Users map is not changed and the error is returned.
func (h *HtpasswdFile) Reload() error {
	r, err := os.Open(h.path)
	if err != nil {
		return err
	}
	defer r.Close()

	cr := csv.NewReader(r)
	cr.Comma = ':'
	cr.Comment = '#'
	cr.TrimLeadingSpace = true

	records, err := cr.ReadAll()
	if err != nil {
		return err
	}
	users := make(map[string]string)
	for _, record := range records {
		users[record[0]] = record[1]
	}

	// Replace the Users map
	h.mutex.Lock()
	h.Users = users
	h.mutex.Unlock()
	return nil
}

// ReloadCheck checks at most once per CheckInterval if the file changed and will reload the file if it did.
// It logs errors and successful reloads, and returns an error if any was encountered.
func (h *HtpasswdFile) ReloadCheck() error {
	select {
	case <-h.throttle:
		stat, err := os.Stat(h.path)
		if err != nil {
			log.Printf("Could not stat htpasswd file: %v", err)
			return err
		}

		reload := false

		h.mutex.Lock()
		if stat.ModTime() != h.stat.ModTime() || stat.Size() != h.stat.Size() {
			reload = true
			h.stat = stat
		}
		h.mutex.Unlock()

		if reload {
			err := h.Reload()
			if err == nil {
				log.Printf("Reloaded htpasswd file")
			} else {
				log.Printf("Could not reload htpasswd file: %v", err)
				return err
			}
		}
	default:
		// No need to check
	}
	return nil
}

// Validate returns true if password matches the stored password for user.  If no password for user is stored, or the
// password is wrong, false is returned.
func (h *HtpasswdFile) Validate(user string, password string) bool {
	_ = h.ReloadCheck()

	h.mutex.Lock()
	realPassword, exists := h.Users[user]
	h.mutex.Unlock()

	if !exists {
		return false
	}
	if realPassword[:5] == "{SHA}" {
		d := sha1.New()
		d.Write([]byte(password))
		if realPassword[5:] == base64.StdEncoding.EncodeToString(d.Sum(nil)) {
			return true
		}
	} else {
		log.Printf("Invalid htpasswd entry for %s. Must be a SHA entry.", user)
	}
	return false
}
