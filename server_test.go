package restserver

import (
	"os"
	"strings"
	"testing"

	"github.com/restic/rest-server/tests"
)

// createOverwriteDeleteSeq returns a sequence which will create a new file at
// path, and then try to overwrite and delete it.
func createOverwriteDeleteSeq(t testing.TB, path string) []tests.TestRequest {
	return []tests.TestRequest{
		{
			Req:      tests.NewRequest(t, "GET", path, nil),
			Expected: []tests.ExpectFunc{tests.ExpectCode(404)},
		},
		{
			Req:      tests.NewRequest(t, "POST", path, strings.NewReader("foobar test config")),
			Expected: []tests.ExpectFunc{tests.ExpectCode(200)},
		},
		{
			Req:      tests.NewRequest(t, "HEAD", path, nil),
			Expected: []tests.ExpectFunc{tests.ExpectCode(200)},
		},
		{
			Req: tests.NewRequest(t, "GET", path, nil),
			Expected: []tests.ExpectFunc{
				tests.ExpectCode(200),
				tests.ExpectBody("foobar test config"),
			},
		},
		{
			Req:      tests.NewRequest(t, "POST", path, strings.NewReader("other config")),
			Expected: []tests.ExpectFunc{tests.ExpectCode(403)},
		},
		{
			Req: tests.NewRequest(t, "GET", path, nil),
			Expected: []tests.ExpectFunc{
				tests.ExpectCode(200),
				tests.ExpectBody("foobar test config"),
			},
		},
		{
			Req:      tests.NewRequest(t, "DELETE", path, nil),
			Expected: []tests.ExpectFunc{tests.ExpectCode(403)},
		},
		{
			Req: tests.NewRequest(t, "GET", path, nil),
			Expected: []tests.ExpectFunc{
				tests.ExpectCode(200),
				tests.ExpectBody("foobar test config"),
			},
		},
	}
}

// TestServer runs tests on the server handler code, especially in append-only mode.
func TestServer(t *testing.T) {
	fsPath, err := tests.NewEmptyFS()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(fsPath)

	server := NewServer(&Config{
		AppendOnly: true,
		Path:       fsPath,
		Debug:      true,
		NoAuth:     true,
	})

	server.testing = true

	go func() {
		if err := server.Run(); err != nil {
			t.Fatal(err)
		}
	}()
	defer func() {
		server.shutdownCh <- struct{}{}
	}()

	// Wait for the serve to fully start
	<-server.startedCh

	// Create repo
	tests.CheckRequest(t, server.Handler.ServeHTTP,
		tests.NewRequest(t, "POST", "/?create=true", nil),
		[]tests.ExpectFunc{tests.ExpectCode(200)},
	)

	randomID := tests.RandomID()
	for _, d := range [][]tests.TestRequest{
		createOverwriteDeleteSeq(t, "/config"),
		createOverwriteDeleteSeq(t, "/data/"+randomID),
		[]tests.TestRequest{
			{
				Req:      tests.NewRequest(t, "GET", "/locks/"+randomID, nil),
				Expected: []tests.ExpectFunc{tests.ExpectCode(404)},
			},
			{
				Req:      tests.NewRequest(t, "POST", "/locks/"+randomID, strings.NewReader("lock file")),
				Expected: []tests.ExpectFunc{tests.ExpectCode(200)},
			},
			{
				Req: tests.NewRequest(t, "GET", "/locks/"+randomID, nil),
				Expected: []tests.ExpectFunc{
					tests.ExpectCode(200),
					tests.ExpectBody("lock file"),
				},
			},
			{
				Req:      tests.NewRequest(t, "POST", "/locks/"+randomID, strings.NewReader("other lock file")),
				Expected: []tests.ExpectFunc{tests.ExpectCode(403)},
			},
			{
				Req:      tests.NewRequest(t, "DELETE", "/locks/"+randomID, nil),
				Expected: []tests.ExpectFunc{tests.ExpectCode(200)},
			},
			{
				Req:      tests.NewRequest(t, "GET", "/locks/"+randomID, nil),
				Expected: []tests.ExpectFunc{tests.ExpectCode(404)},
			},
		},
	} {

		t.Run("", func(t *testing.T) {
			for i, testRequest := range d {
				t.Logf("request %v: %v %v", i, testRequest.Req.Method, testRequest.Req.URL.Path)
				tests.CheckRequest(t, server.Handler.ServeHTTP, testRequest.Req, testRequest.Expected)
			}
		})
	}
}
