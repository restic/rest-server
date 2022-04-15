//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"log"
	"net"

	"github.com/coreos/go-systemd/activation"
)

// findListener tries to find a listener via systemd socket activation. If that
// fails, it tries to create a listener on addr.
func findListener(addr string) (listener net.Listener, err error) {
	// try systemd socket activation
	listeners, err := activation.Listeners()
	if err != nil {
		panic(err)
	}

	switch len(listeners) {
	case 0:
		// no listeners found, listen manually
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("listen on %v failed: %w", addr, err)
		}

		log.Printf("start server on %v", addr)
		return listener, nil

	case 1:
		// one listener supplied by systemd, use that one
		//
		// for testing, run rest-server with systemd-socket-activate as follows:
		//
		//    systemd-socket-activate -l 8080 ./rest-server
		log.Printf("systemd socket activation mode")
		return listeners[0], nil

	default:
		return nil, fmt.Errorf("got %d listeners from systemd, expected one", len(listeners))
	}
}
