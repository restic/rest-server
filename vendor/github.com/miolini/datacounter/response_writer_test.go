package datacounter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriterCounter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}
	req, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	counter := NewResponseWriterCounter(w)
	handler(counter, req)
	if counter.Count() != dataLen {
		t.Fatalf("count mismatch len of test data: %d != %d", counter.Count(), len(data))
	}
}
