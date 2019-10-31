package resticserver

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// CheckInterval represents how often we check for changes in .htpasswd file.
const CheckInterval = 30 * time.Second

// Regex used to find password type. Allowed are SHA and bcrypt.
var (
	SHARegex    = regexp.MustCompile(`^{SHA}`)
	BcryptRegex = regexp.MustCompile(`^\$2b\$|^\$2a\$|^\$2y\$`)
)

// PasswordFile is a map for usernames to passwords.
type PasswordFile struct {
	Users map[string]string // Users contains a map of users and passwords

	mutex    sync.Mutex
	path     string
	stat     os.FileInfo
	throttle chan struct{}
}

// NewPasswordFile reads the users and passwords from a .htpasswd file and returns them.
func NewPasswordFile(path string) (*PasswordFile, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	pass := &PasswordFile{
		mutex:    sync.Mutex{},
		path:     path,
		stat:     stat,
		throttle: make(chan struct{}),
	}

	if err := pass.Reload(); err != nil {
		return nil, err
	}

	// Start a goroutine that limits reload checks to once per CheckInterval
	go pass.throttleTimer()

	return pass, nil
}

// throttleTimer sends at most one message per CheckInterval to throttle file change checks.
func (pass *PasswordFile) throttleTimer() {
	var check struct{}
	for {
		time.Sleep(CheckInterval)
		pass.throttle <- check
	}
}

// Reload reloads the htpasswd file. If the reload fails, the Users map is not changed and the error is returned.
func (pass *PasswordFile) Reload() error {
	r, err := os.Open(pass.path)
	if err != nil {
		return err
	}

	cr := csv.NewReader(r)
	cr.Comma = ':'
	cr.Comment = '#'
	cr.TrimLeadingSpace = true

	records, err := cr.ReadAll()
	if err != nil {
		_ = r.Close()
		return err
	}

	users := make(map[string]string)
	for _, record := range records {
		users[record[0]] = record[1]
	}

	// Replace the users map
	pass.mutex.Lock()
	pass.Users = users
	pass.mutex.Unlock()

	_ = r.Close()
	return nil
}

// ReloadCheck checks at most once per CheckInterval if the file changed and will reload the file if it did.
// It logs errors and successful reloads, and returns an error if any was encountered.
func (pass *PasswordFile) ReloadCheck() error {
	select {
	case <-pass.throttle:
		stat, err := os.Stat(pass.path)
		if err != nil {
			return err
		}

		reload := false

		pass.mutex.Lock()
		if stat.ModTime() != pass.stat.ModTime() || stat.Size() != pass.stat.Size() {
			reload = true
			pass.stat = stat
		}
		pass.mutex.Unlock()

		if reload {
			err := pass.Reload()
			if err == nil {
				return fmt.Errorf("Could not reload htpasswd file: %v", err)
			}
		}
	default:
		// No need to check
	}
	return nil
}

// Validate returns true if password matches the stored password for user.  If no password for user is stored, or the
// password is wrong, false is returned.
func (pass *PasswordFile) Validate(user string, password string) bool {
	_ = pass.ReloadCheck()

	pass.mutex.Lock()
	realPassword, exists := pass.Users[user]
	pass.mutex.Unlock()

	if !exists {
		return false
	}

	switch {
	case SHARegex.MatchString(realPassword):
		d := sha1.New()
		_, _ = d.Write([]byte(password))
		if realPassword[5:] == base64.StdEncoding.EncodeToString(d.Sum(nil)) {
			return true
		}
	case BcryptRegex.MatchString(realPassword):
		err := bcrypt.CompareHashAndPassword([]byte(realPassword), []byte(password))
		if err == nil {
			return true
		}
	}

	return false
}
