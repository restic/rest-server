package tests

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

// RandomID generates a 32 char random id
func RandomID() string {
	buf := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
