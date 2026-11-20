package restserver

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestValidate(t *testing.T) {
	user := "restic"
	pwd := "$2y$05$z/OEmNQamd6m6LSegUErh.r/Owk9Xwmc5lxDheIuHY2Z7XiS6FtJm"
	rawPwd := "test"
	wrongPwd := "wrong"

	tmpfile, err := os.CreateTemp("", "rest-validate-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tmpfile.Write([]byte(user + ":" + pwd + "\n")); err != nil {
		t.Fatal(err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	htpass, err := NewHtpasswdFromFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	for range 10 {
		isValid := htpass.Validate(user, rawPwd)
		if !isValid {
			t.Fatal("correct password not accepted")
		}

		isValid = htpass.Validate(user, wrongPwd)
		if isValid {
			t.Fatal("wrong password accepted")
		}
	}

	if err = os.Remove(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}
}

// writeHtpasswd writes an htpasswd file with the given content to a temporary
// directory and returns its path.
func writeHtpasswd(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".htpasswd")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateUnknownUser(t *testing.T) {
	htpass, err := NewHtpasswdFromFile(writeHtpasswd(t, "restic:$2y$05$z/OEmNQamd6m6LSegUErh.r/Owk9Xwmc5lxDheIuHY2Z7XiS6FtJm\n"))
	if err != nil {
		t.Fatal(err)
	}
	if htpass.Validate("nobody", "test") {
		t.Fatal("unknown user accepted")
	}
	// the dummy hash used for unknown users is as expensive as the real one
	if cost, err := bcrypt.Cost([]byte(htpass.dummyHash)); err != nil || cost != 5 {
		t.Fatalf("dummy hash cost = %d (%v), want 5", cost, err)
	}
}

func TestDummyHashFor(t *testing.T) {
	bcrypt4, err := bcrypt.GenerateFromPassword([]byte("test"), 4)
	if err != nil {
		t.Fatal(err)
	}
	bcrypt5, err := bcrypt.GenerateFromPassword([]byte("test"), 5)
	if err != nil {
		t.Fatal(err)
	}
	sha := "{SHA}qUqP5cyxm6YcTAhz05Hph5gvu9M="

	users := map[string]string{
		"a": string(bcrypt4),
		"b": string(bcrypt5),
		"c": sha,
	}

	// the dummy hash has the cost of the most expensive hash
	dummy, err := dummyHashFor(users, "")
	if err != nil {
		t.Fatal(err)
	}
	if cost, err := bcrypt.Cost([]byte(dummy)); err != nil || cost != 5 {
		t.Fatalf("dummy hash cost = %d (%v), want 5", cost, err)
	}

	// a dummy hash with the right cost is kept
	again, err := dummyHashFor(users, dummy)
	if err != nil {
		t.Fatal(err)
	}
	if again != dummy {
		t.Error("dummy hash with matching cost was not reused")
	}

	// a dummy hash with the wrong cost is replaced
	delete(users, "b")
	changed, err := dummyHashFor(users, dummy)
	if err != nil {
		t.Fatal(err)
	}
	if cost, err := bcrypt.Cost([]byte(changed)); err != nil || cost != 4 {
		t.Fatalf("dummy hash cost = %d (%v), want 4", cost, err)
	}

	// without bcrypt hashes, a SHA hash is used
	shaOnly, err := dummyHashFor(map[string]string{"c": sha}, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !shaRe.MatchString(shaOnly) {
		t.Errorf("dummy hash = %q, want a SHA hash", shaOnly)
	}
	if isMatchingHashAndPassword(shaOnly, "") || isMatchingHashAndPassword(shaOnly, "test") {
		t.Error("dummy SHA hash matches a password")
	}
}
