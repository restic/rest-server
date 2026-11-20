package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestDynamicReload(t *testing.T) {
	cert, key, err := generateCertFiles()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("created %s and %s files", cert, key)
	t.Cleanup(func() {
		_ = os.Remove(cert)
		_ = os.Remove(key)
	})
	err = generateSelfSigned(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	dc, err := newDynamicChecker(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dc.poll(ctx, time.Second)
	crt1Raw, err := dc.getCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	crt1, err := x509.ParseCertificate(crt1Raw.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	err = generateSelfSigned(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 2)
	crt2Raw, err := dc.getCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	crt2, err := x509.ParseCertificate(crt2Raw.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if crt1.SerialNumber.Cmp(crt2.SerialNumber) == 0 {
		t.Fatal("expected certificate to be different")
	}
	t.Logf("cert 1 serial: %s cert 2 serial: %s", crt1.SerialNumber, crt2.SerialNumber)
	// force a certificate
	_ = os.Remove(cert)
	time.Sleep(time.Second * 2)
	crt3Raw, err := dc.getCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	crt3, err := x509.ParseCertificate(crt3Raw.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if crt2.SerialNumber.Cmp(crt3.SerialNumber) != 0 {
		t.Fatal("expected certificate to be certificate")
	}
}

func generateCertFiles() (cert, key string, err error) {
	certFile, err := os.CreateTemp("", "cert")
	if err != nil {
		return "", "", err
	}
	cert = certFile.Name()
	_ = certFile.Close()
	keyFile, err := os.CreateTemp("", "key")
	if err != nil {
		return "", "", err
	}
	key = keyFile.Name()
	_ = keyFile.Close()
	return cert, key, nil
}

var serial = int64(9000)

func NextSerial() *big.Int {
	serial++
	return big.NewInt(serial)
}

func generateSelfSigned(certFile, keyFile string) error {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	template := &x509.Certificate{
		SerialNumber: NextSerial(),
		Subject: pkix.Name{
			Organization: []string{"Widgets Inc"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}
	certDer, err := x509.CreateCertificate(rand.Reader, template, template, pk.Public(), pk)
	if err != nil {
		return err
	}
	keyDer, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return err
	}
	keyFh, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = keyFh.Close()
	}()
	certFh, err := os.OpenFile(certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = certFh.Close()
	}()
	err = pem.Encode(certFh, &pem.Block{Type: "CERTIFICATE", Bytes: certDer})
	if err != nil {
		return err
	}
	err = pem.Encode(keyFh, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer})
	if err != nil {
		return err
	}
	return nil
}
