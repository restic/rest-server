package restserver

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// AuthFailureLogInterval is the minimum time between two log messages about
// rejected authentication attempts from the same client address. Attempts
// within that interval are counted and reported with the next message.
const AuthFailureLogInterval = 10 * time.Second

// maxLoggedUsernameLength bounds the length of a username in a log message.
const maxLoggedUsernameLength = 64

// authFailureLogger logs rejected authentication attempts with the client
// address and the attempted username, so that an operator or a tool like
// fail2ban can notice password guessing. To keep a flood of attempts from
// filling the log, at most one message per client address is written per
// interval. Further attempts are counted and reported with the next message.
type authFailureLogger struct {
	interval time.Duration
	now      func() time.Time
	printf   func(format string, v ...any)

	mu        sync.Mutex
	clients   map[string]*authFailureClient
	lastSweep time.Time
}

type authFailureClient struct {
	user       string    // sanitized username of the last attempt
	lastLog    time.Time // time of the last message about this client
	suppressed int       // attempts since lastLog that were not logged yet
}

func newAuthFailureLogger() *authFailureLogger {
	return &authFailureLogger{
		interval: AuthFailureLogInterval,
		now:      time.Now,
		printf:   log.Printf,
		clients:  make(map[string]*authFailureClient),
	}
}

// Log records a rejected authentication attempt for user from addr.
func (l *authFailureLogger) Log(addr, user string) {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	c, known := l.clients[addr]
	if !known {
		c = &authFailureClient{}
		l.clients[addr] = c
	}
	c.user = sanitizeUsername(user)

	if known && now.Sub(c.lastLog) < l.interval {
		c.suppressed++
		return
	}

	l.report(addr, c)
	c.lastLog = now

	l.sweep(now)
}

// report writes the message for the last attempt of c and the attempts that
// were not logged before it.
func (l *authFailureLogger) report(addr string, c *authFailureClient) {
	if c.suppressed > 0 {
		l.printf("Rejected authentication for user %s from %s (and %d more since the last message)", c.user, addr, c.suppressed)
	} else {
		l.printf("Rejected authentication for user %s from %s", c.user, addr)
	}
	c.suppressed = 0
}

// sweep forgets clients without a logged attempt in the last interval, so
// that the client map does not grow without bounds. Attempts of those clients
// that were counted but not reported yet are reported now. The sweep runs at
// most once per interval.
func (l *authFailureLogger) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.interval {
		return
	}
	l.lastSweep = now

	for addr, c := range l.clients {
		if now.Sub(c.lastLog) < l.interval {
			continue
		}
		if c.suppressed > 0 {
			l.report(addr, c)
		}
		delete(l.clients, addr)
	}
}

// sanitizeUsername returns user quoted and escaped for use in a log message,
// cut to maxLoggedUsernameLength characters. The username is chosen by the
// client and may contain anything.
func sanitizeUsername(user string) string {
	n := 0
	for i := range user {
		if n == maxLoggedUsernameLength {
			user = user[:i] + "..."
			break
		}
		n++
	}
	return strconv.Quote(user)
}

// clientAddress returns the IP address of the client that sent r. With
// --proxy-headers, this is the address reported by the proxy.
func clientAddress(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
