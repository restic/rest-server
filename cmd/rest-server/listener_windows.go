package main

import (
	"fmt"
	"log"
	"net"
)

// findListener creates a listener.
func findListener(addr string) (listener net.Listener, err error) {
	// listen manually
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %v failed: %w", addr, err)
	}

	log.Printf("start server on %v", listener.Addr())
	return listener, nil
}
