package main

import (
	"context"
	"crypto/tls"
	"io/fs"
	"log"
	"os"
	"sync/atomic"
	"time"
)

type dynamicChecker struct {
	certificate               atomic.Pointer[tls.Certificate]
	keyFile, certFile         string
	keyFileInfo, certFileInfo fs.FileInfo
}

// newDynamicChecker creates a struct that holds the data we need to do
// dynamic certificate reloads from disk.  If it cannot load the files
// or they are invalid, an error is returned.  Following a successful
// instantiation, the getCertificate method will always return a valid
// certificate, and we should call the poll method to check for changes.
func newDynamicChecker(certFile, keyFile string) (*dynamicChecker, error) {
	keyFileInfo, err := os.Stat(keyFile)
	if err != nil {
		return nil, err
	}
	certFileInfo, err := os.Stat(certFile)
	if err != nil {
		return nil, err
	}
	crt, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	dc := &dynamicChecker{
		keyFile:      keyFile,
		certFile:     certFile,
		keyFileInfo:  keyFileInfo,
		certFileInfo: certFileInfo,
	}
	dc.certificate.Store(&crt)
	return dc, nil
}

// getCertificate - always returns a valid tls.Certificate and nil error.
func (dc *dynamicChecker) getCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return dc.certificate.Load(), nil
}

// poll runs in a goroutine and periodically polls the key and cert for
// updates.
func (dc *dynamicChecker) poll(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTimer(interval)
		defer t.Stop() // go >= 1.23 means we don't have to check the return
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				keyFileInfo, err := os.Stat(dc.keyFile)
				if err != nil {
					log.Printf("could not stat keyFile %s, using previous cert: %s", dc.keyFile, err)
					break // select
				}
				certFileInfo, err := os.Stat(dc.certFile)
				if err != nil {
					log.Printf("could not stat certFile %s, using previous cert: %s", dc.certFile, err)
					break // select
				}
				if !keyFileInfo.ModTime().Equal(dc.keyFileInfo.ModTime()) ||
					keyFileInfo.Size() != dc.keyFileInfo.Size() ||
					!certFileInfo.ModTime().Equal(dc.certFileInfo.ModTime()) ||
					certFileInfo.Size() != dc.certFileInfo.Size() {
					// they changed on disk, reload
					crt, err := tls.LoadX509KeyPair(dc.certFile, dc.keyFile)
					if err != nil {
						log.Printf("could not load cert and key files, using previous cert: %s", err)
						break // select
					}
					dc.certificate.Store(&crt)
					dc.certFileInfo = certFileInfo
					dc.keyFileInfo = keyFileInfo
					log.Printf("successfully reloaded certificate from disk")
				}
			} // end select
			t.Reset(interval)
		} // end for
	}()
}
