package datacounter

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

var data = []byte("Hello, World!")
var dataLen = uint64(len(data))

func TestReaderCounter(t *testing.T) {
	buf := bytes.Buffer{}
	buf.Write(data)
	counter := NewReaderCounter(&buf)
	io.Copy(ioutil.Discard, counter)
	if counter.Count() != dataLen {
		t.Fatalf("count mismatch len of test data: %d != %d", counter.Count(), len(data))
	}
}
