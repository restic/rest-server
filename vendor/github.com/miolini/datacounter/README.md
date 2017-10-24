# datacounter
Golang counters for readers/writers

[![Build Status](https://travis-ci.org/miolini/datacounter.svg)](https://travis-ci.org/miolini/datacounter) [![GoDoc](https://godoc.org/github.com/miolini/datacounter?status.svg)](http://godoc.org/github.com/miolini/datacounter) [![Go Report Card](http://goreportcard.com/badge/miolini/datacounter)](http://goreportcard.com/report/miolini/datacounter)

## Examples 
### ReaderCounter
```go
buf := bytes.Buffer{}
buf.Write(data)
counter := datacounter.NewReaderCounter(&buf)
io.Copy(ioutil.Discard, counter)
if counter.Count() != dataLen {
	t.Fatalf("count mismatch len of test data: %d != %d", counter.Count(), len(data))
}`
```
### WriterCounter
```go
buf := bytes.Buffer{}
counter := datacounter.NewWriterCounter(&buf)
counter.Write(data)
if counter.Count() != dataLen {
	t.Fatalf("count mismatch len of test data: %d != %d", counter.Count(), len(data))
}
```
### http.ResponseWriter Counter
```go
handler := func(w http.ResponseWriter, r *http.Request) {
	w.Write(data)
}
req, err := http.NewRequest("GET", "http://example.com/foo", nil)
if err != nil {
	t.Fatal(err)
}
w := httptest.NewRecorder()
counter := datacounter.NewResponseWriterCounter(w)
handler(counter, req)
if counter.Count() != dataLen {
	t.Fatalf("count mismatch len of test data: %d != %d", counter.Count(), len(data))
}
```
