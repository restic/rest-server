package datacounter

import (
	"bytes"
	"testing"
)

func TestWriterCounter(t *testing.T) {
	buf := bytes.Buffer{}
	counter := NewWriterCounter(&buf)
	counter.Write(data)
	if counter.Count() != dataLen {
		t.Fatalf("count mismatch len of test data: %d != %d", counter.Count(), len(data))
	}
}
