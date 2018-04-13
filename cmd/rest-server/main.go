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

var (
	repoPath     = "/tmp/restic"
	listen       = ":8000"
	logFile      string
	cpuProfile   string
	tlsKey       string
	tlsCert      string
	useTLS       bool
	noAuth       bool
	appendOnly   bool
	privateRepos bool
	prometheus   bool
	debug        bool
	showVersion  bool
)

func init() {
	flags := cmdRoot.Flags()
	flags.StringVar(&cpuProfile, "cpu-profile", cpuProfile, "write CPU profile to file")
	flags.BoolVar(&debug, "debug", debug, "output debug messages")
	flags.StringVar(&listen, "listen", listen, "listen address")
	flags.StringVar(&logFile, "log", logFile, "log HTTP requests in the combined log format")
	flags.StringVar(&repoPath, "path", repoPath, "data directory")
	flags.BoolVar(&useTLS, "tls", useTLS, "turn on TLS support")
	flags.StringVar(&tlsCert, "tls-cert", tlsCert, "TLS certificate path")
	flags.StringVar(&tlsKey, "tls-key", tlsKey, "TLS key path")
	flags.BoolVar(&noAuth, "no-auth", noAuth, "disable .htpasswd authentication")
	flags.BoolVar(&appendOnly, "append-only", appendOnly, "enable append only mode")
	flags.BoolVar(&privateRepos, "private-repos", privateRepos, "users can only access their private repo")
	flags.BoolVar(&prometheus, "prometheus", prometheus, "enable Prometheus metrics")
	flags.BoolVarP(&showVersion, "version", "V", showVersion, "output version and exit")
}

var version = "manually"

func tlsSettings() (bool, string, string, error) {
	var key, cert string
	if !useTLS && (tlsKey != "" || tlsCert != "") {
		return false, "", "", errors.New("requires enabled TLS")
	} else if !useTLS {
		return false, "", "", nil
	}
	if tlsKey != "" {
		key = tlsKey
	} else {
		key = filepath.Join(repoPath, "private_key")
	}
	if tlsCert != "" {
		cert = tlsCert
	} else {
		cert = filepath.Join(repoPath, "public_key")
	}
	return useTLS, key, cert, nil
}

func getHandler(config restserver.Config) (http.Handler, error) {
	mux := restserver.NewHandler(config)
	if config.NoAuth {
		log.Println("Authentication disabled")
		return mux, nil
	}

	log.Println("Authentication enabled")
	htpasswdFile, err := restserver.NewHtpasswdFromFile(filepath.Join(config.Path, ".htpasswd"))
	if err != nil {
		return nil, fmt.Errorf("cannot load .htpasswd (use --no-auth to disable): %v", err)
	}
	return config.AuthHandler(htpasswdFile, mux), nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Printf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	log.SetFlags(0)

	log.Printf("Data directory: %s", repoPath)

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

	config := restserver.Config{
		Path:         repoPath,
		Listen:       listen,
		Log:          logFile,
		CPUProfile:   cpuProfile,
		TLSKey:       tlsKey,
		TLSCert:      tlsCert,
		TLS:          useTLS,
		NoAuth:       noAuth,
		AppendOnly:   appendOnly,
		PrivateRepos: privateRepos,
		Prometheus:   prometheus,
		Debug:        debug,
	}

	handler, err := getHandler(config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if privateRepos {
		log.Println("Private repositories enabled")
	} else {
		log.Println("Private repositories disabled")
	}

	enabledTLS, privateKey, publicKey, err := tlsSettings()
	if err != nil {
		return err
	}
	if !enabledTLS {
		log.Printf("Starting server on %s\n", listen)
		err = http.ListenAndServe(listen, handler)
	} else {

		log.Println("TLS enabled")
		log.Printf("Private key: %s", privateKey)
		log.Printf("Public key(certificate): %s", publicKey)
		log.Printf("Starting server on %s\n", listen)
		err = http.ListenAndServeTLS(listen, publicKey, privateKey, handler)
	}

	return err
}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
