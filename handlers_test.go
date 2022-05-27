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

// TestSuite is a group of TestRequest that covers some functionality
type TestSuite []struct {
	seq []TestRequest
}

const (
	GetForbidden = 1 << iota
	PostForbidden
	PostBrokenForbidden
	DeleteForbidden
)

// createOverwriteDeleteSeq returns a sequence which will create a new file at
// path, and then try to overwrite and delete it if allowed by flags
func createOverwriteDeleteSeq(t testing.TB, path string, data string, forbiddenFlags int) []TestRequest {
	// path, read it and then try to overwrite and delete (if not forbidden by flags)
	checkFlag := func(flag int, flagged []wantFunc, arg []wantFunc) []wantFunc {
		if flag&forbiddenFlags == 0 {
			return arg
		}
		return flagged
	}

	checkForbidden := func(flag int, arg []wantFunc) []wantFunc {
		return checkFlag(flag, []wantFunc{
			wantCode(http.StatusForbidden),
		}, arg)
	}

	ifNotDeleted := func(arg []wantFunc) []wantFunc {
		if forbiddenFlags&DeleteForbidden != 0 {
			return arg
		}
		return []wantFunc{
			wantCode(http.StatusNotFound),
		}
	}

	brokenData := data + "_broken"
	expectedData := data
	if forbiddenFlags&PostBrokenForbidden == 0 {
		expectedData = brokenData
	}

	// add a file, try to overwrite and delete it
	req := []TestRequest{
		{
			req:  newRequest(t, "GET", path, nil),
			want: checkForbidden(GetForbidden, []wantFunc{wantCode(http.StatusNotFound)}),
		},
		{
			// broken upload must fail if repo is configured to verify blobs
			req: newRequest(t, "POST", path, strings.NewReader(brokenData)),
			want: checkForbidden(PostForbidden,
				checkFlag(PostBrokenForbidden,
					[]wantFunc{wantCode(http.StatusBadRequest)},
					[]wantFunc{wantCode(http.StatusOK)})),
		},
		{ // if blob verification is not enabled, we'll get Forbidden here because broken data was uploaded before
			req: newRequest(t, "POST", path, strings.NewReader(data)),
			want: checkForbidden(PostForbidden,
				checkFlag(PostBrokenForbidden,
					[]wantFunc{wantCode(http.StatusOK)},
					[]wantFunc{wantCode(http.StatusForbidden)})),
		},
		{
			req: newRequest(t, "GET", path, nil),
			want: checkForbidden(GetForbidden, []wantFunc{
				wantCode(http.StatusOK),
				wantBody(expectedData),
			}),
		},
		{ // always Forbidden because it's overwrite of existing data
			req:  newRequest(t, "POST", path, strings.NewReader(data+"other stuff")),
			want: []wantFunc{wantCode(http.StatusForbidden)},
		},
		{
			req: newRequest(t, "GET", path, nil),
			want: checkForbidden(GetForbidden, []wantFunc{
				wantCode(http.StatusOK),
				wantBody(expectedData),
			}),
		},
		{
			req:  newRequest(t, "DELETE", path, nil),
			want: checkForbidden(DeleteForbidden, []wantFunc{wantCode(http.StatusOK)}),
		},
		{
			req: newRequest(t, "GET", path, nil),
			want: checkForbidden(GetForbidden, ifNotDeleted([]wantFunc{
				wantCode(http.StatusOK),
				wantBody(expectedData),
			})),
		},
	}
	return req
}

func randomDataAndId(t *testing.T) (string, string) {
	buf := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		t.Fatal(err)
	}
	data := "random data file " + hex.EncodeToString(buf)
	dataHash := sha256.Sum256([]byte(data))
	fileID := hex.EncodeToString(dataHash[:])
	return data, fileID
}

// testResticHandler creates repo in temporary dir and runs tests on the restic handler code
func testResticHandler(t *testing.T, tests *TestSuite, server Server, pathsToCreate []string) {
	// setup the server with a local backend in a temporary directory
	tempdir, err := ioutil.TempDir("", "rest-server-test-")
	if err != nil {
		t.Fatal(err)
	}

	// make sure the tempdir is properly removed
	defer func() {
		err := os.RemoveAll(tempdir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	server.Path = tempdir
	mux, err := NewHandler(&server)
	if err != nil {
		t.Fatalf("error from NewHandler: %v", err)
	}

	// create the repos
	for _, path := range pathsToCreate {
		checkRequest(t, mux.ServeHTTP,
			newRequest(t, "POST", path+"?create=true", nil),
			[]wantFunc{wantCode(http.StatusOK)})
	}

	for _, test := range *tests {
		t.Run("", func(t *testing.T) {
			for i, seq := range test.seq {
				t.Logf("request %v: %v %v", i, seq.req.Method, seq.req.URL.Path)
				checkRequest(t, mux.ServeHTTP, seq.req, seq.want)
			}
		})
	}
}

// TestResticHandler runs tests on the restic handler code, default mode (everything allowed)
func TestResticDefaultHandler(t *testing.T) {
	data, fileID := randomDataAndId(t)

	var tests = TestSuite{
		{createOverwriteDeleteSeq(t, "/config", data, 0)},
		{createOverwriteDeleteSeq(t, "/keys/"+fileID, data, PostBrokenForbidden)},
		{createOverwriteDeleteSeq(t, "/index/"+fileID, data, PostBrokenForbidden)},
		{createOverwriteDeleteSeq(t, "/data/"+fileID, data, PostBrokenForbidden)},
		{createOverwriteDeleteSeq(t, "/snapshots/"+fileID, data, PostBrokenForbidden)},
		{createOverwriteDeleteSeq(t, "/locks/"+fileID, data, PostBrokenForbidden)},
	}
	// set append-only mode
	testResticHandler(t, &tests, Server{
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
	}, []string{"/"})
}

// TestResticHandler runs tests on the restic handler code, disabled blob verification
func TestResticNoVerifyUploadHandler(t *testing.T) {
	data, fileID := randomDataAndId(t)

	var tests = TestSuite{
		{createOverwriteDeleteSeq(t, "/config", data, 0)},
		{createOverwriteDeleteSeq(t, "/keys/"+fileID, data, 0)},
		{createOverwriteDeleteSeq(t, "/index/"+fileID, data, 0)},
		{createOverwriteDeleteSeq(t, "/data/"+fileID, data, 0)},
		{createOverwriteDeleteSeq(t, "/snapshots/"+fileID, data, 0)},
		{createOverwriteDeleteSeq(t, "/locks/"+fileID, data, 0)},
	}
	// set append-only mode
	testResticHandler(t, &tests, Server{
		NoAuth:         true,
		Debug:          true,
		PanicOnError:   true,
		NoVerifyUpload: true,
	}, []string{"/"})
}

// TestResticHandler runs tests on the restic handler code, default mode (everything allowed)
func TestResticAppendOnlyUploadHandler(t *testing.T) {
	data, fileID := randomDataAndId(t)

	var tests = TestSuite{
		{createOverwriteDeleteSeq(t, "/config", data, DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/keys/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/index/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/data/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/snapshots/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/locks/"+fileID, data, PostBrokenForbidden)},
	}
	// set append-only mode
	testResticHandler(t, &tests, Server{
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
		AppendOnly:   true,
	}, []string{"/"})
}

// TestResticHandler runs tests on the restic handler code, default mode (everything allowed)
func TestResticWriteOnlyUploadHandler(t *testing.T) {
	data, fileID := randomDataAndId(t)

	var tests = TestSuite{
		{createOverwriteDeleteSeq(t, "/config", data, DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/keys/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/index/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/data/"+fileID, data, GetForbidden|PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/snapshots/"+fileID, data, PostBrokenForbidden|DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/locks/"+fileID, data, PostBrokenForbidden)},
	}
	// set append-only mode
	testResticHandler(t, &tests, Server{
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
		WriteOnly:    true,
	}, []string{"/"})
}

// TestResticHandler runs tests on the restic handler code, default mode (everything allowed)
func TestResticPrivateRepoUploadHandler(t *testing.T) {
	data, fileID := randomDataAndId(t)

	var tests = TestSuite{
		{createOverwriteDeleteSeq(t, "/parent1/sub1/config", "foobar", DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/parent1/sub1/data/"+fileID, data, DeleteForbidden|PostBrokenForbidden)},
		{createOverwriteDeleteSeq(t, "/parent1/config", "foobar", DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/parent1/data/"+fileID, data, DeleteForbidden|PostBrokenForbidden)},
		{createOverwriteDeleteSeq(t, "/parent2/config", "foobar", DeleteForbidden)},
		{createOverwriteDeleteSeq(t, "/parent2/data/"+fileID, data, DeleteForbidden|PostBrokenForbidden)},
	}
	// set append-only mode
	testResticHandler(t, &tests, Server{
		AppendOnly:   true,
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
	}, []string{"/", "/parent1/sub1/", "/parent1/", "/parent2/"})
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
	// setup the server with a local backend in a temporary directory
	tempdir, err := ioutil.TempDir("", "rest-server-test-")
	if err != nil {
		t.Fatal(err)
	}

	// make sure the tempdir is properly removed
	defer func() {
		err := os.RemoveAll(tempdir)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// configure path, the race condition doesn't happen for append-only repositories
	mux, err := NewHandler(&Server{
		AppendOnly:   false,
		Path:         tempdir,
		NoAuth:       true,
		Debug:        true,
		PanicOnError: true,
	})
	if err != nil {
		t.Fatalf("error from NewHandler: %v", err)
	}

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
