package tests

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ExpectFunc tests the HTTP response in res and calls t.Error() if something is incorrect.
type ExpectFunc func(t testing.TB, res *httptest.ResponseRecorder)

// NewRequest returns a new HTTP request with the given params. On error, t.Fatal is called.
func NewRequest(t testing.TB, method, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// ExpectCode returns a function which checks that the response has the correct HTTP status code.
func ExpectCode(code int) ExpectFunc {
	return func(t testing.TB, res *httptest.ResponseRecorder) {
		if res.Code != code {
			t.Errorf("expected response code '%v', found '%v'", code, res.Code)
		}
	}
}

// ExpectBody returns a function which checks that the response has the data in the body.
func ExpectBody(body string) ExpectFunc {
	return func(t testing.TB, res *httptest.ResponseRecorder) {
		if res.Body == nil {
			t.Errorf("expected body '%q', found 'nil'", body)
			return
		}

		if !bytes.Equal(res.Body.Bytes(), []byte(body)) {
			t.Errorf("expected body: '%q', found: '%q'", body, res.Body.Bytes())
		}
	}
}

// CheckRequest uses f to process the request and runs the checker functions on the result.
func CheckRequest(t testing.TB, f http.HandlerFunc, req *http.Request, expect []ExpectFunc) {
	rr := httptest.NewRecorder()
	f(rr, req)

	for _, fn := range expect {
		fn(t, rr)
	}
}

// TestRequest is a HTTP request with (optional) tests for the response.
type TestRequest struct {
	Req      *http.Request
	Expected []ExpectFunc
}
