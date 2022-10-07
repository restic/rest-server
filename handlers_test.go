package restserver

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/minio/sha256-simd"
)

func TestJoin(t *testing.T) {
	var tests = []struct {
		base   string
		names  []string
		result string
	}{
		{"/", []string{"foo", "bar"}, "/foo/bar"},
		{"/srv/server", []string{"foo", "bar"}, "/srv/server/foo/bar"},
		{"/srv/server", []string{"foo", "..", "bar"}, "/srv/server/foo/bar"},
		{"/srv/server", []string{"..", "bar"}, "/srv/server/bar"},
		{"/srv/server", []string{".."}, "/srv/server"},
		{"/srv/server", []string{"..", ".."}, "/srv/server"},
		{"/srv/server", []string{"repo", "data"}, "/srv/server/repo/data"},
		{"/srv/server", []string{"repo", "data", "..", ".."}, "/srv/server/repo/data"},
		{"/srv/server", []string{"repo", "data", "..", "data", "..", "..", ".."}, "/srv/server/repo/data/data"},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := join(filepath.FromSlash(test.base), test.names...)
			if err != nil {
				t.Fatal(err)
			}

			want := filepath.FromSlash(test.result)
			if got != want {
				t.Fatalf("wrong result returned, want %v, got %v", want, got)
			}
		})
	}
}

// declare a few helper functions

// wantFunc tests the HTTP response in res and calls t.Error() if something is incorrect.
type wantFunc func(t testing.TB, res *httptest.ResponseRecorder)

// newRequest returns a new HTTP request with the given params. On error, t.Fatal is called.
func newRequest(t testing.TB, method, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// wantCode returns a function which checks that the response has the correct HTTP status code.
func wantCode(code int) wantFunc {
	return func(t testing.TB, res *httptest.ResponseRecorder) {
		t.Helper()
		if res.Code != code {
			t.Errorf("wrong response code, want %v, got %v", code, res.Code)
		}
	}
}

// wantBody returns a function which checks that the response has the data in the body.
func wantBody(body string) wantFunc {
	return func(t testing.TB, res *httptest.ResponseRecorder) {
		t.Helper()
		if res.Body == nil {
			t.Errorf("body is nil, want %q", body)
			return
		}

		if !bytes.Equal(res.Body.Bytes(), []byte(body)) {
			t.Errorf("wrong response body, want:\n  %q\ngot:\n  %q", body, res.Body.Bytes())
		}
	}
}

// checkRequest uses f to process the request and runs the checker functions on the result.
func checkRequest(t testing.TB, f http.HandlerFunc, req *http.Request, want []wantFunc) {
	t.Helper()
	rr := httptest.NewRecorder()
	f(rr, req)

	for _, fn := range want {
		fn(t, rr)
	}
}

// TestRequest is a sequence of HTTP requests with (optional) tests for the response.
type TestRequest struct {
	req  *http.Request
	want []wantFunc
}

// createOverwriteDeleteSeq returns a sequence which will create a new file at
// path, and then try to overwrite and delete it.
func createOverwriteDeleteSeq(t testing.TB, path string, data string) []TestRequest {
	// add a file, try to overwrite and delete it
	req := []TestRequest{
		{
			req:  newRequest(t, "GET", path, nil),
			want: []wantFunc{wantCode(http.StatusNotFound)},
		},
	}

	if !strings.HasSuffix(path, "/config") {
		req = append(req, TestRequest{
			// broken upload must fail
			req:  newRequest(t, "POST", path, strings.NewReader(data+"broken")),
			want: []wantFunc{wantCode(http.StatusBadRequest)},
		})
	}

	req = append(req,
		TestRequest{
			req:  newRequest(t, "POST", path, strings.NewReader(data)),
			want: []wantFunc{wantCode(http.StatusOK)},
		},
		TestRequest{
			req: newRequest(t, "GET", path, nil),
			want: []wantFunc{
				wantCode(http.StatusOK),
				wantBody(data),
			},
		},
		TestRequest{
			req:  newRequest(t, "POST", path, strings.NewReader(data+"other stuff")),
			want: []wantFunc{wantCode(http.StatusForbidden)},
		},
		TestRequest{
			req: newRequest(t, "GET", path, nil),
			want: []wantFunc{
				wantCode(http.StatusOK),
				wantBody(data),
			},
		},
		TestRequest{
			req:  newRequest(t, "DELETE", path, nil),
			want: []wantFunc{wantCode(http.StatusForbidden)},
		},
		TestRequest{
			req: newRequest(t, "GET", path, nil),
			want: []wantFunc{
				wantCode(http.StatusOK),
				wantBody(data),
			},
		},
	)
	return req
}

func createTestHandler(t *testing.T, conf Server) (http.Handler, string, string, string, func()) {
	buf := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		t.Fatal(err)
	}
	data := "random data file " + hex.EncodeToString(buf)
	dataHash := sha256.Sum256([]byte(data))
	fileID := hex.EncodeToString(dataHash[:])

	// setup the server with a local backend in a temporary directory
	tempdir, err := ioutil.TempDir("", "rest-server-test-")
	if err != nil {
		t.Fatal(err)
	}

	// make sure the tempdir is properly removed
	cleanup := func() {
		err := os.RemoveAll(tempdir)
		if err != nil {
			t.Fatal(err)
		}
	}

	conf.Path = tempdir
	mux, err := NewHandler(&conf)
	if err != nil {
		t.Fatalf("error from NewHandler: %v", err)
	}
	return mux, data, fileID, tempdir, cleanup
}

// TestResticAppendOnlyHandler runs tests on the restic handler code, especially in append-only mode.
func TestResticAppendOnlyHandler(t *testing.T) {
	mux, data, fileID, _, cleanup := createTestHandler(t, Server{
		AppendOnly:   true,
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
	})
	defer cleanup()

	var tests = []struct {
		seq []TestRequest
	}{
		{createOverwriteDeleteSeq(t, "/config", data)},
		{createOverwriteDeleteSeq(t, "/data/"+fileID, data)},
		{
			// ensure we can add and remove lock files
			[]TestRequest{
				{
					req:  newRequest(t, "GET", "/locks/"+fileID, nil),
					want: []wantFunc{wantCode(http.StatusNotFound)},
				},
				{
					req:  newRequest(t, "POST", "/locks/"+fileID, strings.NewReader(data+"broken")),
					want: []wantFunc{wantCode(http.StatusBadRequest)},
				},
				{
					req:  newRequest(t, "POST", "/locks/"+fileID, strings.NewReader(data)),
					want: []wantFunc{wantCode(http.StatusOK)},
				},
				{
					req: newRequest(t, "GET", "/locks/"+fileID, nil),
					want: []wantFunc{
						wantCode(http.StatusOK),
						wantBody(data),
					},
				},
				{
					req:  newRequest(t, "POST", "/locks/"+fileID, strings.NewReader(data+"other data")),
					want: []wantFunc{wantCode(http.StatusForbidden)},
				},
				{
					req:  newRequest(t, "DELETE", "/locks/"+fileID, nil),
					want: []wantFunc{wantCode(http.StatusOK)},
				},
				{
					req:  newRequest(t, "GET", "/locks/"+fileID, nil),
					want: []wantFunc{wantCode(http.StatusNotFound)},
				},
			},
		},

		// Test subrepos
		{createOverwriteDeleteSeq(t, "/parent1/sub1/config", "foobar")},
		{createOverwriteDeleteSeq(t, "/parent1/sub1/data/"+fileID, data)},
		{createOverwriteDeleteSeq(t, "/parent1/config", "foobar")},
		{createOverwriteDeleteSeq(t, "/parent1/data/"+fileID, data)},
		{createOverwriteDeleteSeq(t, "/parent2/config", "foobar")},
		{createOverwriteDeleteSeq(t, "/parent2/data/"+fileID, data)},
	}

	// create the repos
	for _, path := range []string{"/", "/parent1/sub1/", "/parent1/", "/parent2/"} {
		checkRequest(t, mux.ServeHTTP,
			newRequest(t, "POST", path+"?create=true", nil),
			[]wantFunc{wantCode(http.StatusOK)})
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			for i, seq := range test.seq {
				t.Logf("request %v: %v %v", i, seq.req.Method, seq.req.URL.Path)
				checkRequest(t, mux.ServeHTTP, seq.req, seq.want)
			}
		})
	}
}

// createOverwriteDeleteSeq returns a sequence which will create a new file at
// path, and then deletes it twice.
func createIdempotentDeleteSeq(t testing.TB, path string, data string) []TestRequest {
	return []TestRequest{
		{
			req:  newRequest(t, "POST", path, strings.NewReader(data)),
			want: []wantFunc{wantCode(http.StatusOK)},
		},
		{
			req:  newRequest(t, "DELETE", path, nil),
			want: []wantFunc{wantCode(http.StatusOK)},
		},
		{
			req:  newRequest(t, "GET", path, nil),
			want: []wantFunc{wantCode(http.StatusNotFound)},
		},
		{
			req:  newRequest(t, "DELETE", path, nil),
			want: []wantFunc{wantCode(http.StatusOK)},
		},
	}
}

// TestResticHandler runs tests on the restic handler code, especially in append-only mode.
func TestResticHandler(t *testing.T) {
	mux, data, fileID, _, cleanup := createTestHandler(t, Server{
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
	})
	defer cleanup()

	var tests = []struct {
		seq []TestRequest
	}{
		{createIdempotentDeleteSeq(t, "/config", data)},
		{createIdempotentDeleteSeq(t, "/data/"+fileID, data)},
	}

	// create the repo
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "POST", "/?create=true", nil),
		[]wantFunc{wantCode(http.StatusOK)})

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			for i, seq := range test.seq {
				t.Logf("request %v: %v %v", i, seq.req.Method, seq.req.URL.Path)
				checkRequest(t, mux.ServeHTTP, seq.req, seq.want)
			}
		})
	}
}

// TestResticErrorHandler runs tests on the restic handler error handling.
func TestResticErrorHandler(t *testing.T) {
	mux, _, _, tempdir, cleanup := createTestHandler(t, Server{
		AppendOnly: true,
		NoAuth:     true,
		Debug:      true,
	})
	defer cleanup()

	var tests = []struct {
		seq []TestRequest
	}{
		// Test inaccessible file
		{
			[]TestRequest{{
				req:  newRequest(t, "GET", "/config", nil),
				want: []wantFunc{wantCode(http.StatusInternalServerError)},
			}},
		},
		{
			[]TestRequest{{
				req:  newRequest(t, "GET", "/parent4/config", nil),
				want: []wantFunc{wantCode(http.StatusNotFound)},
			}},
		},
	}

	// create the repo
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "POST", "/?create=true", nil),
		[]wantFunc{wantCode(http.StatusOK)})
	// create inaccessible config
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "POST", "/config", strings.NewReader("example")),
		[]wantFunc{wantCode(http.StatusOK)})
	err := os.Chmod(path.Join(tempdir, "config"), 0o000)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			for i, seq := range test.seq {
				t.Logf("request %v: %v %v", i, seq.req.Method, seq.req.URL.Path)
				checkRequest(t, mux.ServeHTTP, seq.req, seq.want)
			}
		})
	}
}

func TestSplitURLPath(t *testing.T) {
	var tests = []struct {
		// Params
		urlPath  string
		maxDepth int
		// Expected result
		folderPath []string
		remainder  string
	}{
		{"/", 0, nil, "/"},
		{"/", 2, nil, "/"},
		{"/foo/bar/locks/0123", 0, nil, "/foo/bar/locks/0123"},
		{"/foo/bar/locks/0123", 1, []string{"foo"}, "/bar/locks/0123"},
		{"/foo/bar/locks/0123", 2, []string{"foo", "bar"}, "/locks/0123"},
		{"/foo/bar/locks/0123", 3, []string{"foo", "bar"}, "/locks/0123"},
		{"/foo/bar/zzz/locks/0123", 2, []string{"foo", "bar"}, "/zzz/locks/0123"},
		{"/foo/bar/zzz/locks/0123", 3, []string{"foo", "bar", "zzz"}, "/locks/0123"},
		{"/foo/bar/locks/", 2, []string{"foo", "bar"}, "/locks/"},
		{"/foo/locks/", 2, []string{"foo"}, "/locks/"},
		{"/foo/data/", 2, []string{"foo"}, "/data/"},
		{"/foo/index/", 2, []string{"foo"}, "/index/"},
		{"/foo/keys/", 2, []string{"foo"}, "/keys/"},
		{"/foo/snapshots/", 2, []string{"foo"}, "/snapshots/"},
		{"/foo/config", 2, []string{"foo"}, "/config"},
		{"/foo/", 2, []string{"foo"}, "/"},
		{"/foo/bar/", 2, []string{"foo", "bar"}, "/"},
		{"/foo/bar", 2, []string{"foo"}, "/bar"},
		{"/locks/", 2, nil, "/locks/"},
		// This function only splits, it does not check the path components!
		{"/././locks/", 2, []string{".", "."}, "/locks/"},
		{"/../../locks/", 2, []string{"..", ".."}, "/locks/"},
		{"///locks/", 2, []string{"", ""}, "/locks/"},
		{"////locks/", 2, []string{"", ""}, "//locks/"},
		// Robustness against broken input
		{"/", -42, nil, "/"},
		{"foo", 2, nil, "foo"},
		{"foo/bar", 2, nil, "foo/bar"},
		{"", 2, nil, ""},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			folderPath, remainder := splitURLPath(test.urlPath, test.maxDepth)

			var fpEqual bool
			if len(test.folderPath) == 0 && len(folderPath) == 0 {
				fpEqual = true // this check allows for nil vs empty slice
			} else {
				fpEqual = reflect.DeepEqual(test.folderPath, folderPath)
			}
			if !fpEqual {
				t.Errorf("wrong folderPath: want %v, got %v", test.folderPath, folderPath)
			}

			if test.remainder != remainder {
				t.Errorf("wrong remainder: want %v, got %v", test.remainder, remainder)
			}
		})
	}
}

// delayErrorReader blocks until Continue is closed, closes the channel FirstRead and then returns Err.
type delayErrorReader struct {
	FirstRead     chan struct{}
	firstReadOnce sync.Once

	Err error

	Continue chan struct{}
}

func newDelayedErrorReader(err error) *delayErrorReader {
	return &delayErrorReader{
		Err:       err,
		Continue:  make(chan struct{}),
		FirstRead: make(chan struct{}),
	}
}

func (d *delayErrorReader) Read(p []byte) (int, error) {
	d.firstReadOnce.Do(func() {
		// close the channel to signal that the first read has happened
		close(d.FirstRead)
	})
	<-d.Continue
	return 0, d.Err
}

// TestAbortedRequest runs tests with concurrent upload requests for the same file.
func TestAbortedRequest(t *testing.T) {
	// the race condition doesn't happen for append-only repositories
	mux, _, _, _, cleanup := createTestHandler(t, Server{
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
	})
	defer cleanup()

	// create the repo
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "POST", "/?create=true", nil),
		[]wantFunc{wantCode(http.StatusOK)})

	var (
		id = "b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"
		wg sync.WaitGroup
	)

	// the first request is an upload to a file which blocks while reading the
	// body and then after some data returns an error
	rd := newDelayedErrorReader(io.ErrUnexpectedEOF)

	wg.Add(1)
	go func() {
		defer wg.Done()

		// first, read some string, then read from rd (which blocks and then
		// returns an error)
		dataReader := io.MultiReader(strings.NewReader("invalid data from aborted request\n"), rd)

		t.Logf("start first upload")
		req := newRequest(t, "POST", "/data/"+id, dataReader)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		t.Logf("first upload done, response %v (%v)", rr.Code, rr.Result().Status)
	}()

	// wait until the first request starts reading from the body
	<-rd.FirstRead

	// then while the first request is blocked we send a second request to
	// delete the file and a third request to upload to the file again, only
	// then the first request is unblocked.

	t.Logf("delete file")
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "DELETE", "/data/"+id, nil),
		nil) // don't check anything, restic also ignores errors here

	t.Logf("upload again")
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "POST", "/data/"+id, strings.NewReader("foo\n")),
		[]wantFunc{wantCode(http.StatusOK)})

	// unblock the reader for the first request now so it can continue
	close(rd.Continue)

	// wait for the first request to continue
	wg.Wait()

	// request the file again, it must exist and contain the string from the
	// second request
	checkRequest(t, mux.ServeHTTP,
		newRequest(t, "GET", "/data/"+id, nil),
		[]wantFunc{
			wantCode(http.StatusOK),
			wantBody("foo\n"),
		},
	)
}
