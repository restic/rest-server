package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	restserver "github.com/restic/rest-server"
	"github.com/spf13/cobra"
)

// cmdRoot is the base command when no other command has been specified.
var cmdRoot = &cobra.Command{
	Use:           "rest-server",
	Short:         "Run a REST server for use with restic",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runRoot,
	// Use this instead of other --version code when the Cobra dependency can be updated.
	//Version:       fmt.Sprintf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH),
}

var server = restserver.Server{
	Path:   "/tmp/restic",
	Listen: ":8000",
}

var (
	showVersion bool
	cpuProfile  string
)

func init() {
	flags := cmdRoot.Flags()
	flags.StringVar(&cpuProfile, "cpu-profile", cpuProfile, "write CPU profile to file")
	flags.BoolVar(&server.Debug, "debug", server.Debug, "output debug messages")
	flags.StringVar(&server.Listen, "listen", server.Listen, "listen address")
	flags.StringVar(&server.Log, "log", server.Log, "log HTTP requests in the combined log format")
	flags.Int64Var(&server.MaxRepoSize, "max-size", server.MaxRepoSize, "the maximum size of the repository in bytes")
	flags.StringVar(&server.Path, "path", server.Path, "data directory")
	flags.BoolVar(&server.TLS, "tls", server.TLS, "turn on TLS support")
	flags.StringVar(&server.TLSCert, "tls-cert", server.TLSCert, "TLS certificate path")
	flags.StringVar(&server.TLSKey, "tls-key", server.TLSKey, "TLS key path")
	flags.StringVar(&server.TLSClientCerts, "tls-client-certs", server.TLSClientCerts, "PEM encoded client certificates for TLS authentication")
	flags.BoolVar(&server.NoAuth, "no-auth", server.NoAuth, "disable .htpasswd authentication")
	flags.BoolVar(&server.AppendOnly, "append-only", server.AppendOnly, "enable append only mode")
	flags.BoolVar(&server.PrivateRepos, "private-repos", server.PrivateRepos, "users can only access their private repo")
	flags.BoolVar(&server.Prometheus, "prometheus", server.Prometheus, "enable Prometheus metrics")
	flags.BoolVarP(&showVersion, "version", "V", showVersion, "output version and exit")
}

var version = "manually"

func tlsSettings() (bool, string, string, string, error) {
	var key, cert string
	if !server.TLS && (server.TLSKey != "" || server.TLSCert != "") {
		return false, "", "", "", errors.New("requires enabled TLS")
	} else if !server.TLS {
		return false, "", "", "", nil
	}
	if server.TLSKey != "" {
		key = server.TLSKey
	} else {
		key = filepath.Join(server.Path, "private_key")
	}
	if server.TLSCert != "" {
		cert = server.TLSCert
	} else {
		cert = filepath.Join(server.Path, "public_key")
	}
	return server.TLS, key, cert, server.TLSClientCerts, nil
}

func addClientCerts(p *x509.CertPool, fp string) error {
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return err
	}

	if !p.AppendCertsFromPEM(data) {
		return errors.New("PEM file contains invalid certificate")
	}

	return nil
}

func listenAndServeTLS(addr, certFile, keyFile, clientCertsFile string, handler http.Handler) error {
	httpServer := http.Server{Addr: addr, Handler: handler}
	if clientCertsFile != "" {
		clientCerts := x509.NewCertPool()
		err := addClientCerts(clientCerts, clientCertsFile)
		if err != nil {
			return err
		}

		authType := tls.VerifyClientCertIfGiven
		if server.NoAuth {
			authType = tls.RequireAndVerifyClientCert
		}
		httpServer.TLSConfig = &tls.Config{
			ClientCAs:  clientCerts,
			ClientAuth: authType,
		}
	}

	return httpServer.ListenAndServeTLS(certFile, keyFile)
}

func getHandler(server restserver.Server) (http.Handler, error) {
	mux := restserver.NewHandler(server)
	if server.NoAuth {
		log.Println("Authentication disabled")
		return mux, nil
	}

	log.Println("Authentication enabled")
	htpasswdFile, err := restserver.NewHtpasswdFromFile(filepath.Join(server.Path, ".htpasswd"))
	if err != nil {
		return nil, fmt.Errorf("cannot load .htpasswd (use --no-auth to disable): %v", err)
	}
	return server.AuthHandler(htpasswdFile, mux), nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Printf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	log.SetFlags(0)

	log.Printf("Data directory: %s", server.Path)

	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}
		log.Println("CPU profiling enabled")
		defer pprof.StopCPUProfile()
	}

	handler, err := getHandler(server)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if server.PrivateRepos {
		log.Println("Private repositories enabled")
	} else {
		log.Println("Private repositories disabled")
	}

	enabledTLS, privateKey, publicKey, clientCerts, err := tlsSettings()
	if err != nil {
		return err
	}
	if !enabledTLS {
		log.Printf("Starting server on %s\n", server.Listen)
		err = http.ListenAndServe(server.Listen, handler)
	} else {

		log.Println("TLS enabled")
		log.Printf("Private key: %s", privateKey)
		log.Printf("Public key(certificate): %s", publicKey)
		log.Printf("Starting server on %s\n", server.Listen)
		err = listenAndServeTLS(server.Listen,
			publicKey, privateKey, clientCerts, handler)
	}

	return err
}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
