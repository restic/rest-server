package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	restserver "github.com/restic/rest-server"
)

func TestTLSSettings(t *testing.T) {
	type expected struct {
		TLSKey  string
		TLSCert string
		Error   bool
	}
	type passed struct {
		Path    string
		TLS     bool
		TLSKey  string
		TLSCert string
	}

	var tests = []struct {
		passed   passed
		expected expected
	}{
		{passed{TLS: false}, expected{"", "", false}},
		{passed{TLS: true}, expected{
			filepath.Join(os.TempDir(), "restic/private_key"),
			filepath.Join(os.TempDir(), "restic/public_key"),
			false,
		}},
		{passed{
			Path: os.TempDir(),
			TLS:  true,
		}, expected{
			filepath.Join(os.TempDir(), "private_key"),
			filepath.Join(os.TempDir(), "public_key"),
			false,
		}},
		{passed{Path: os.TempDir(), TLS: true, TLSKey: "/etc/restic/key", TLSCert: "/etc/restic/cert"}, expected{"/etc/restic/key", "/etc/restic/cert", false}},
		{passed{Path: os.TempDir(), TLS: false, TLSKey: "/etc/restic/key", TLSCert: "/etc/restic/cert"}, expected{"", "", true}},
		{passed{Path: os.TempDir(), TLS: false, TLSKey: "/etc/restic/key"}, expected{"", "", true}},
		{passed{Path: os.TempDir(), TLS: false, TLSCert: "/etc/restic/cert"}, expected{"", "", true}},
	}

	for _, test := range tests {
		app := newRestServerApp()
		t.Run("", func(t *testing.T) {
			// defer func() { restserver.Server = defaultConfig }()
			if test.passed.Path != "" {
				app.Server.Path = test.passed.Path
			}
			app.Server.TLS = test.passed.TLS
			app.Server.TLSKey = test.passed.TLSKey
			app.Server.TLSCert = test.passed.TLSCert

			gotTLS, gotKey, gotCert, err := app.tlsSettings()
			if err != nil && !test.expected.Error {
				t.Fatalf("tls_settings returned err (%v)", err)
			}
			if test.expected.Error {
				if err == nil {
					t.Fatalf("Error not returned properly (%v)", test)
				} else {
					return
				}
			}
			if gotTLS != test.passed.TLS {
				t.Errorf("TLS enabled, want (%v), got (%v)", test.passed.TLS, gotTLS)
			}
			wantKey := test.expected.TLSKey
			if gotKey != wantKey {
				t.Errorf("wrong TLSPrivPath path, want (%v), got (%v)", wantKey, gotKey)
			}

			wantCert := test.expected.TLSCert
			if gotCert != wantCert {
				t.Errorf("wrong TLSCertPath path, want (%v), got (%v)", wantCert, gotCert)
			}

		})
	}
}

func TestGetHandler(t *testing.T) {
	dir, err := os.MkdirTemp("", "rest-server-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(dir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	getHandler := restserver.NewHandler

	// With NoAuth = false and no .htpasswd
	_, err = getHandler(&restserver.Server{Path: dir})
	if err == nil {
		t.Errorf("NoAuth=false: expected error, got nil")
	}

	// With NoAuth = true and no .htpasswd
	_, err = getHandler(&restserver.Server{NoAuth: true, Path: dir})
	if err != nil {
		t.Errorf("NoAuth=true: expected no error, got %v", err)
	}

	// With NoAuth = false and custom .htpasswd
	htpFile, err := os.CreateTemp(dir, "custom")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(htpFile.Name())
		if err != nil {
			t.Fatal(err)
		}
	}()
	_, err = getHandler(&restserver.Server{HtpasswdPath: htpFile.Name()})
	if err != nil {
		t.Errorf("NoAuth=false with custom htpasswd: expected no error, got %v", err)
	}

	// Create .htpasswd
	htpasswd := filepath.Join(dir, ".htpasswd")
	err = os.WriteFile(htpasswd, []byte(""), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.Remove(htpasswd)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// With NoAuth = false and with .htpasswd
	_, err = getHandler(&restserver.Server{Path: dir})
	if err != nil {
		t.Errorf("NoAuth=false with .htpasswd: expected no error, got %v", err)
	}
}

// helper method to test the app. Starts app with passed arguments,
// then will call the callback function which can make requests against
// the application. If the callback function fails due to errors returned
// by http.Do() (i.e. *url.Error), then it will be retried until successful,
// or the passed timeout passes.
func testServerWithArgs(args []string, timeout time.Duration, cb func(context.Context, *restServerApp) error) error {
	// create the app with passed args
	app := newRestServerApp()
	app.CmdRoot.SetArgs(args)

	// create context that will timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// wait group for our client and server tasks
	jobs := &sync.WaitGroup{}
	jobs.Add(2)

	// run the server, saving the error
	var serverErr error
	go func() {
		defer jobs.Done()
		defer cancel() // if the server is stopped, no point keep the client alive
		serverErr = app.CmdRoot.ExecuteContext(ctx)
	}()

	// run the client, saving the error
	var clientErr error
	go func() {
		defer jobs.Done()
		defer cancel() // once the client is done, stop the server

		var urlError *url.Error

		// execute in loop, as we will retry for network errors
		// (such as the server hasn't started yet)
		for {
			clientErr = cb(ctx, app)
			switch {
			case clientErr == nil:
				return // success, we're done
			case errors.As(clientErr, &urlError):
				// if a network error (url.Error), then wait and retry
				// as server may not be ready yet
				select {
				case <-time.After(time.Millisecond * 100):
					continue
				case <-ctx.Done(): // unless we run out of time first
					clientErr = context.Canceled
					return
				}
			default:
				return // other error type, we're done
			}
		}
	}()

	// wait for both to complete
	jobs.Wait()

	// report back if either failed
	if clientErr != nil || serverErr != nil {
		return fmt.Errorf("client or server error, client: %v, server: %v", clientErr, serverErr)
	}

	return nil
}

func TestHttpListen(t *testing.T) {
	td := t.TempDir()

	// create some content and parent dirs
	if err := os.MkdirAll(filepath.Join(td, "data", "repo1"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "data", "repo1", "config"), []byte("foo"), 0700); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"--no-auth", "--path", filepath.Join(td, "data"), "--listen", "127.0.0.1:0"},    // test emphemeral port
		{"--no-auth", "--path", filepath.Join(td, "data"), "--listen", "127.0.0.1:9000"}, // test "normal" port
		{"--no-auth", "--path", filepath.Join(td, "data"), "--listen", "127.0.0.1:9000"}, // test that server was shutdown cleanly and that we can re-use that port
	} {
		err := testServerWithArgs(args, time.Second*10, func(ctx context.Context, app *restServerApp) error {
			for _, test := range []struct {
				Path       string
				StatusCode int
			}{
				{"/repo1/", http.StatusMethodNotAllowed},
				{"/repo1/config", http.StatusOK},
				{"/repo2/config", http.StatusNotFound},
			} {
				listenAddr := app.ListenerAddress()
				if listenAddr == nil {
					return &url.Error{} // return this type of err, as we know this will retry
				}
				port := strings.Split(listenAddr.String(), ":")[1]

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%s%s", port, test.Path), nil)
				if err != nil {
					return err
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}
				err = resp.Body.Close()
				if err != nil {
					return err
				}
				if resp.StatusCode != test.StatusCode {
					return fmt.Errorf("expected %d from server, instead got %d (path %s)", test.StatusCode, resp.StatusCode, test.Path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
