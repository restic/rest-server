package main

import (
	"errors"
	restserver "github.com/restic/rest-server"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

// cmdRoot is the base command when no other command has been specified.
var cmdRoot = &cobra.Command{
	Use:           "rest-server",
	Short:         "Run a REST server for use with restic",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runRoot,
}

func init() {
	flags := cmdRoot.Flags()
	flags.StringVar(&restserver.Config.CPUProfile, "cpuprofile", restserver.Config.CPUProfile, "write CPU profile to file")
	flags.BoolVar(&restserver.Config.Debug, "debug", restserver.Config.Debug, "output debug messages")
	flags.StringVar(&restserver.Config.Listen, "listen", restserver.Config.Listen, "listen address")
	flags.StringVar(&restserver.Config.Log, "log", restserver.Config.Log, "log HTTP requests in the combined log format")
	flags.StringVar(&restserver.Config.Path, "path", restserver.Config.Path, "data directory")
	flags.BoolVar(&restserver.Config.TLS, "tls", restserver.Config.TLS, "turn on TLS support")
	flags.StringVar(&restserver.Config.TLSCert, "tls-cert", restserver.Config.TLSCert, "TLS certificate path")
	flags.StringVar(&restserver.Config.TLSKey, "tls-key", restserver.Config.TLSKey, "TLS key path")
	flags.BoolVar(&restserver.Config.AppendOnly, "append-only", restserver.Config.AppendOnly, "enable append only mode")
	flags.BoolVar(&restserver.Config.Prometheus, "prometheus", restserver.Config.Prometheus, "enable Prometheus metrics")
}

var version = "manually"

func tlsSettings() (bool, string, string, error) {
	var key, cert string
	enabledTLS := restserver.Config.TLS
	if !enabledTLS && (restserver.Config.TLSKey != "" || restserver.Config.TLSCert != "") {
		return false, "", "", errors.New("requires enabled TLS")
	} else if !enabledTLS {
		return false, "", "", nil
	}
	if restserver.Config.TLSKey != "" {
		key = restserver.Config.TLSKey
	} else {
		key = filepath.Join(restserver.Config.Path, "private_key")
	}
	if restserver.Config.TLSCert != "" {
		cert = restserver.Config.TLSCert
	} else {
		cert = filepath.Join(restserver.Config.Path, "public_key")
	}
	return enabledTLS, key, cert, nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	log.SetFlags(0)

	log.Printf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("Data directory: %s", restserver.Config.Path)

	if restserver.Config.CPUProfile != "" {
		f, err := os.Create(restserver.Config.CPUProfile)
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}
		log.Println("CPU profiling enabled")
		defer pprof.StopCPUProfile()
	}

	mux := restserver.NewMux()

	var handler http.Handler
	htpasswdFile, err := restserver.NewHtpasswdFromFile(filepath.Join(restserver.Config.Path, ".htpasswd"))
	if err != nil {
		handler = mux
		log.Println("Authentication disabled")
	} else {
		handler = restserver.AuthHandler(htpasswdFile, mux)
		log.Println("Authentication enabled")
	}

	enabledTLS, privateKey, publicKey, err := tlsSettings()
	if err != nil {
		return err
	}
	if !enabledTLS {
		log.Printf("Starting server on %s\n", restserver.Config.Listen)
		err = http.ListenAndServe(restserver.Config.Listen, handler)
	} else {

		log.Println("TLS enabled")
		log.Printf("Private key: %s", privateKey)
		log.Printf("Public key(certificate): %s", publicKey)
		log.Printf("Starting server on %s\n", restserver.Config.Listen)
		err = http.ListenAndServeTLS(restserver.Config.Listen, publicKey, privateKey, handler)
	}

	return err
}
func main() {
	if err := cmdRoot.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
