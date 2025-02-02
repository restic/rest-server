//go:build !windows
// +build !windows

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnixSocket(t *testing.T) {
	td := t.TempDir()

	// this is the socket we'll listen on and connect to
	tempSocket := filepath.Join(td, "sock")

	// create some content and parent dirs
	if err := os.MkdirAll(filepath.Join(td, "data", "repo1"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "data", "repo1", "config"), []byte("foo"), 0700); err != nil {
		t.Fatal(err)
	}

	// run the following twice, to test that the server will
	// cleanup its socket file when quitting, which won't happen
	// if it doesn't exit gracefully
	for i := 0; i < 2; i++ {
		err := testServerWithArgs([]string{
			"--no-auth",
			"--path", filepath.Join(td, "data"),
			"--listen", fmt.Sprintf("unix:%s", tempSocket),
		}, time.Second, func(ctx context.Context, _ *restServerApp) error {
			// custom client that will talk HTTP to unix socket
			client := http.Client{
				Transport: &http.Transport{
					DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
						return net.Dial("unix", tempSocket)
					},
				},
			}
			for _, test := range []struct {
				Path       string
				StatusCode int
			}{
				{"/repo1/", http.StatusMethodNotAllowed},
				{"/repo1/config", http.StatusOK},
				{"/repo2/config", http.StatusNotFound},
			} {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ignored"+test.Path, nil)
				if err != nil {
					return err
				}
				resp, err := client.Do(req)
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
