package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"syscall"

	restserver "github.com/restic/rest-server"
	"github.com/spf13/cobra"
)

type restServerApp struct {
	CmdRoot    *cobra.Command
	Server     restserver.Server
	CPUProfile string

	listenerAddressMu sync.Mutex
	listenerAddress   net.Addr // set after startup
}

// cmdRoot is the base command when no other command has been specified.
func newRestServerApp() *restServerApp {
	rv := &restServerApp{
		CmdRoot: &cobra.Command{
			Use:           "rest-server",
			Short:         "Run a REST server for use with restic",
			SilenceErrors: true,
			SilenceUsage:  true,
			Args: func(_ *cobra.Command, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("rest-server expects no arguments - unknown argument: %s", args[0])
				}
				return nil
			},
			Version: fmt.Sprintf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		Server: restserver.Server{
			Path:      filepath.Join(os.TempDir(), "restic"),
			Listen:    ":8000",
			TLSMinVer: "1.2",
		},
	}
	rv.CmdRoot.RunE = rv.runRoot
	flags := rv.CmdRoot.Flags()

	flags.StringVar(&rv.CPUProfile, "cpu-profile", rv.CPUProfile, "write CPU profile to file")
	flags.BoolVar(&rv.Server.Debug, "debug", rv.Server.Debug, "output debug messages")
	flags.StringVar(&rv.Server.Listen, "listen", rv.Server.Listen, "listen address")
	flags.StringVar(&rv.Server.Log, "log", rv.Server.Log, "write HTTP requests in the combined log format to the specified `filename` (use \"-\" for logging to stdout)")
	flags.Int64Var(&rv.Server.MaxRepoSize, "max-size", rv.Server.MaxRepoSize, "the maximum size of the repository in bytes")
	flags.StringVar(&rv.Server.Path, "path", rv.Server.Path, "data directory")
	flags.BoolVar(&rv.Server.TLS, "tls", rv.Server.TLS, "turn on TLS support")
	flags.StringVar(&rv.Server.TLSCert, "tls-cert", rv.Server.TLSCert, "TLS certificate path")
	flags.StringVar(&rv.Server.TLSKey, "tls-key", rv.Server.TLSKey, "TLS key path")
	flags.StringVar(&rv.Server.TLSMinVer, "tls-min-ver", rv.Server.TLSMinVer, "TLS min version (default: 1.2)")
	flags.BoolVar(&rv.Server.NoAuth, "no-auth", rv.Server.NoAuth, "disable authentication")
	flags.StringVar(&rv.Server.HtpasswdPath, "htpasswd-file", rv.Server.HtpasswdPath, "location of .htpasswd file (default: \"<data directory>/.htpasswd)\"")
	flags.StringVar(&rv.Server.ProxyAuthUsername, "proxy-auth-username", rv.Server.ProxyAuthUsername, "specifies the HTTP header containing the username for proxy-based authentication")
	flags.BoolVar(&rv.Server.NoVerifyUpload, "no-verify-upload", rv.Server.NoVerifyUpload,
		"do not verify the integrity of uploaded data. DO NOT enable unless the rest-server runs on a very low-power device")
	flags.BoolVar(&rv.Server.AppendOnly, "append-only", rv.Server.AppendOnly, "enable append only mode")
	flags.BoolVar(&rv.Server.PrivateRepos, "private-repos", rv.Server.PrivateRepos, "users can only access their private repo")
	flags.BoolVar(&rv.Server.Prometheus, "prometheus", rv.Server.Prometheus, "enable Prometheus metrics")
	flags.BoolVar(&rv.Server.PrometheusNoAuth, "prometheus-no-auth", rv.Server.PrometheusNoAuth, "disable auth for Prometheus /metrics endpoint")
	flags.BoolVar(&rv.Server.GroupAccessibleRepos, "group-accessible-repos", rv.Server.GroupAccessibleRepos, "let filesystem group be able to access repo files")

	return rv
}

var version = "0.13.0"

func (app *restServerApp) tlsSettings() (bool, string, string, error) {
	var key, cert string
	if !app.Server.TLS && (app.Server.TLSKey != "" || app.Server.TLSCert != "") {
		return false, "", "", errors.New("requires enabled TLS")
	} else if !app.Server.TLS {
		return false, "", "", nil
	}
	if app.Server.TLSKey != "" {
		key = app.Server.TLSKey
	} else {
		key = filepath.Join(app.Server.Path, "private_key")
	}
	if app.Server.TLSCert != "" {
		cert = app.Server.TLSCert
	} else {
		cert = filepath.Join(app.Server.Path, "public_key")
	}
	return app.Server.TLS, key, cert, nil
}

// returns the address that the app is listening on.
// returns nil if the application hasn't finished starting yet
func (app *restServerApp) ListenerAddress() net.Addr {
	app.listenerAddressMu.Lock()
	defer app.listenerAddressMu.Unlock()
	return app.listenerAddress
}

func (app *restServerApp) runRoot(_ *cobra.Command, _ []string) error {
	log.SetFlags(0)

	log.Printf("Data directory: %s", app.Server.Path)

	if app.CPUProfile != "" {
		f, err := os.Create(app.CPUProfile)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()

		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()

		log.Println("CPU profiling enabled")
		defer log.Println("Stopped CPU profiling")
	}

	if app.Server.NoAuth {
		log.Println("Authentication disabled")
	} else {
		if app.Server.ProxyAuthUsername == "" {
			log.Println("Authentication enabled")
		} else {
			log.Println("Proxy Authentication enabled.")
		}
	}

	handler, err := restserver.NewHandler(&app.Server)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if app.Server.AppendOnly {
		log.Println("Append only mode enabled")
	} else {
		log.Println("Append only mode disabled")
	}

	if app.Server.PrivateRepos {
		log.Println("Private repositories enabled")
	} else {
		log.Println("Private repositories disabled")
	}

	if app.Server.GroupAccessibleRepos {
		log.Println("Group accessible repos enabled")
	} else {
		log.Println("Group accessible repos disabled")
	}

	enabledTLS, privateKey, publicKey, err := app.tlsSettings()
	if err != nil {
		return err
	}

	listener, err := findListener(app.Server.Listen)
	if err != nil {
		return fmt.Errorf("unable to listen: %w", err)
	}

	// set listener address, this is useful for tests
	app.listenerAddressMu.Lock()
	app.listenerAddress = listener.Addr()
	app.listenerAddressMu.Unlock()

	tlscfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}
	switch app.Server.TLSMinVer {
	case "1.2":
		tlscfg.MinVersion = tls.VersionTLS12
	case "1.3":
		tlscfg.MinVersion = tls.VersionTLS13
	default:
		return fmt.Errorf("Unsupported TLS min version: %s", app.Server.TLSMinVer)
	}

	srv := &http.Server{
		Handler:   handler,
		TLSConfig: tlscfg,
	}

	// run server in background
	go func() {
		if !enabledTLS {
			err = srv.Serve(listener)
		} else {
			log.Printf("TLS enabled, private key %s, pubkey %v", privateKey, publicKey)
			err = srv.ServeTLS(listener, publicKey, privateKey)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve returned err: %v", err)
		}
	}()

	// wait until done
	<-app.CmdRoot.Context().Done()

	// gracefully shutdown server
	if err := srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("server shutdown returned an err: %w", err)
	}

	log.Println("shutdown cleanly")
	return nil
}

func main() {
	// create context to be notified on interrupt or term signal so that we can shutdown cleanly
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := newRestServerApp().CmdRoot.ExecuteContext(ctx); err != nil {
		log.Fatalf("error: %v", err)
	}
}
