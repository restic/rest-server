package restserver

import (
	"fmt"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

// testAuthFailureLogger returns a logger with a controllable clock that
// collects its messages instead of writing them to the log.
func testAuthFailureLogger() (l *authFailureLogger, now *time.Time, lines *[]string) {
	t := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	now = &t
	lines = new([]string)

	l = newAuthFailureLogger()
	l.now = func() time.Time { return *now }
	l.printf = func(format string, v ...any) {
		*lines = append(*lines, fmt.Sprintf(format, v...))
	}
	return l, now, lines
}

func TestAuthFailureLogger(t *testing.T) {
	l, now, lines := testAuthFailureLogger()

	check := func(want ...string) {
		t.Helper()
		if !slices.Equal(*lines, want) {
			t.Errorf("log messages:\n%s\nwant:\n%s", strings.Join(*lines, "\n"), strings.Join(want, "\n"))
		}
		*lines = nil
	}

	// the first attempt of a client is logged, further ones within the
	// interval are only counted, other clients are independent
	l.Log("192.0.2.1", "alice")
	l.Log("192.0.2.1", "alice")
	l.Log("192.0.2.1", "bob")
	l.Log("192.0.2.2", "alice")
	check(
		`Rejected authentication for user "alice" from 192.0.2.1`,
		`Rejected authentication for user "alice" from 192.0.2.2`,
	)

	// after the interval, the next attempt reports the counted ones
	*now = now.Add(l.interval)
	l.Log("192.0.2.1", "carol")
	check(`Rejected authentication for user "carol" from 192.0.2.1 (and 2 more since the last message)`)

	// the count was reset
	*now = now.Add(l.interval)
	l.Log("192.0.2.1", "carol")
	check(`Rejected authentication for user "carol" from 192.0.2.1`)

	// a client that is idle for an interval is forgotten by the sweep;
	// attempts that were counted but not reported are reported first
	l.Log("192.0.2.1", "dave")
	*now = now.Add(2 * l.interval)
	l.Log("192.0.2.3", "erin")
	check(
		`Rejected authentication for user "erin" from 192.0.2.3`,
		`Rejected authentication for user "dave" from 192.0.2.1 (and 1 more since the last message)`,
	)
	if got := len(l.clients); got != 1 {
		t.Errorf("tracked clients = %d, want 1", got)
	}
}

func TestSanitizeUsername(t *testing.T) {
	long := strings.Repeat("ä", maxLoggedUsernameLength)
	tests := []struct {
		user string
		want string
	}{
		{"alice", `"alice"`},
		{"", `""`},
		{"a\nb \"c\" \x00", `"a\nb \"c\" \x00"`},
		{long, `"` + long + `"`},
		{long + "x", `"` + long + `..."`},
	}
	for _, tt := range tests {
		if got := sanitizeUsername(tt.user); got != tt.want {
			t.Errorf("sanitizeUsername(%q) = %s, want %s", tt.user, got, tt.want)
		}
	}
}

func TestClientAddress(t *testing.T) {
	tests := []struct {
		remoteAddr string
		want       string
	}{
		{"192.0.2.1:51234", "192.0.2.1"},
		{"[2001:db8::1]:51234", "2001:db8::1"},
		// --proxy-headers replaces the remote address by the bare address
		// found in the proxy headers
		{"192.0.2.1", "192.0.2.1"},
		{"2001:db8::1", "2001:db8::1"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tt.remoteAddr
		if got := clientAddress(req); got != tt.want {
			t.Errorf("clientAddress(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
		}
	}
}
