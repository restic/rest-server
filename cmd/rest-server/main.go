package main

import (
	"errors"
	"fmt"
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

func init() {
	flags := cmdRoot.Flags()
	flags.StringVar(&restserver.Config.CPUProfile, "cpu-profile", restserver.Config.CPUProfile, "write CPU profile to file")
	flags.BoolVar(&restserver.Config.Debug, "debug", restserver.Config.Debug, "output debug messages")
	flags.StringVar(&restserver.Config.Listen, "listen", restserver.Config.Listen, "listen address")
	flags.StringVar(&restserver.Config.Log, "log", restserver.Config.Log, "log HTTP requests in the combined log format")
	flags.StringVar(&restserver.Config.Path, "path", restserver.Config.Path, "data directory")
	flags.BoolVar(&restserver.Config.TLS, "tls", restserver.Config.TLS, "turn on TLS support")
	flags.StringVar(&restserver.Config.TLSCert, "tls-cert", restserver.Config.TLSCert, "TLS certificate path")
	flags.StringVar(&restserver.Config.TLSKey, "tls-key", restserver.Config.TLSKey, "TLS key path")
	flags.BoolVar(&restserver.Config.NoAuth, "no-auth", restserver.Config.NoAuth, "disable .htpasswd authentication")
	flags.BoolVar(&restserver.Config.AppendOnly, "append-only", restserver.Config.AppendOnly, "enable append only mode")
	flags.BoolVar(&restserver.Config.PrivateRepos, "private-repos", restserver.Config.PrivateRepos, "users can only access their private repo")
	flags.BoolVar(&restserver.Config.Prometheus, "prometheus", restserver.Config.Prometheus, "enable Prometheus metrics")
	flags.BoolVarP(&restserver.Config.Version, "version", "V", restserver.Config.Version, "output version and exit")
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

func getHandler() (http.Handler, error) {
	mux := restserver.NewMux()
	if restserver.Config.NoAuth {
		log.Println("Authentication disabled")
		return mux, nil
	}

	log.Println("Authentication enabled")
	htpasswdFile, err := restserver.NewHtpasswdFromFile(filepath.Join(restserver.Config.Path, ".htpasswd"))
	if err != nil {
		return nil, fmt.Errorf("cannot load .htpasswd (use --no-auth to disable): %v", err)
	}
	return restserver.AuthHandler(htpasswdFile, mux), nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	if restserver.Config.Version {
		fmt.Printf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	log.SetFlags(0)

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

	handler, err := getHandler()
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if restserver.Config.PrivateRepos {
		log.Println("Private repositories enabled")
	} else {
		log.Println("Private repositories disabled")
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
