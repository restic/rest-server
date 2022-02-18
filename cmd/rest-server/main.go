package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"syscall"

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
	Version:       fmt.Sprintf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH),
}

var server = restserver.Server{
	Path:   filepath.Join(os.TempDir(), "restic"),
	Listen: ":8000",
}

var (
	cpuProfile string
)

func init() {
	flags := cmdRoot.Flags()
	flags.StringVar(&cpuProfile, "cpu-profile", cpuProfile, "write CPU profile to file")
	flags.BoolVar(&server.Debug, "debug", server.Debug, "output debug messages")
	flags.StringVar(&server.Listen, "listen", server.Listen, "listen address")
	flags.StringVar(&server.Log, "log", server.Log, "write HTTP requests in the combined log format to the specified `filename`")
	flags.Int64Var(&server.MaxRepoSize, "max-size", server.MaxRepoSize, "the maximum size of the repository in bytes")
	flags.StringVar(&server.Path, "path", server.Path, "data directory")
	flags.BoolVar(&server.TLS, "tls", server.TLS, "turn on TLS support")
	flags.StringVar(&server.TLSCert, "tls-cert", server.TLSCert, "TLS certificate path")
	flags.StringVar(&server.TLSKey, "tls-key", server.TLSKey, "TLS key path")
	flags.BoolVar(&server.NoAuth, "no-auth", server.NoAuth, "disable .htpasswd authentication")
	flags.BoolVar(&server.NoVerifyUpload, "no-verify-upload", server.NoVerifyUpload,
		"do not verify the integrity of uploaded data. DO NOT enable unless the rest-server runs on a very low-power device")
	flags.BoolVar(&server.AppendOnly, "append-only", server.AppendOnly, "enable append only mode")
	flags.BoolVar(&server.PrivateRepos, "private-repos", server.PrivateRepos, "users can only access their private repo")
	flags.BoolVar(&server.Prometheus, "prometheus", server.Prometheus, "enable Prometheus metrics")
	flags.BoolVar(&server.PrometheusNoAuth, "prometheus-no-auth", server.PrometheusNoAuth, "disable auth for Prometheus /metrics endpoint")
}

var version = "0.11.0"

func tlsSettings() (bool, string, string, error) {
	var key, cert string
	if !server.TLS && (server.TLSKey != "" || server.TLSCert != "") {
		return false, "", "", errors.New("requires enabled TLS")
	} else if !server.TLS {
		return false, "", "", nil
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
	return server.TLS, key, cert, nil
}

func runRoot(cmd *cobra.Command, args []string) error {
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

		// clean profiling shutdown on sigint
		sigintCh := make(chan os.Signal, 1)
		go func() {
			for range sigintCh {
				pprof.StopCPUProfile()
				log.Println("Stopped CPU profiling")
				err := f.Close()
				if err != nil {
					log.Printf("error closing CPU profile file: %v", err)
				}
				os.Exit(130)
			}
		}()
		signal.Notify(sigintCh, syscall.SIGINT)
	}

	if server.NoAuth {
		log.Println("Authentication disabled")
	} else {
		log.Println("Authentication enabled")
	}

	handler, err := restserver.NewHandler(&server)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if server.PrivateRepos {
		log.Println("Private repositories enabled")
	} else {
		log.Println("Private repositories disabled")
	}

	enabledTLS, privateKey, publicKey, err := tlsSettings()
	if err != nil {
		return err
	}

	listener, err := findListener(server.Listen)
	if err != nil {
		return fmt.Errorf("unable to listen: %w", err)
	}

	if !enabledTLS {
		err = http.Serve(listener, handler)
	} else {
		log.Printf("TLS enabled, private key %s, pubkey %v", privateKey, publicKey)
		err = http.ServeTLS(listener, handler, publicKey, privateKey)
	}

	return err
}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
