package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/spf13/cobra"
	"goji.io"
	"goji.io/pat"
)

// cmdRoot is the base command when no other command has been specified.
var cmdRoot = &cobra.Command{
	Use:           "rest-server",
	Short:         "Run a REST server for use with restic",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          runRoot,
}

var config = struct {
	path       string
	listen     string
	tls        bool
	cpuprofile string
	debug      bool
}{}

func init() {
	flags := cmdRoot.Flags()
	flags.StringVar(&config.path, "path", "/tmp/restic", "data directory")
	flags.StringVar(&config.listen, "listen", ":8000", "listen address")
	flags.BoolVar(&config.tls, "tls", false, "turn on TLS support")
	flags.StringVar(&config.cpuprofile, "cpuprofile", "", "write CPU profile to file")
	flags.BoolVar(&config.debug, "debug", false, "output debug messages")
}

func debugHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s", r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
}

func setupMux() *goji.Mux {
	mux := goji.NewMux()

	if config.debug {
		mux.Use(debugHandler)
	}

	mux.HandleFunc(pat.Head("/config"), CheckConfig)
	mux.HandleFunc(pat.Head("/:repo/config"), CheckConfig)
	mux.HandleFunc(pat.Get("/config"), GetConfig)
	mux.HandleFunc(pat.Get("/:repo/config"), GetConfig)
	mux.HandleFunc(pat.Post("/config"), SaveConfig)
	mux.HandleFunc(pat.Post("/:repo/config"), SaveConfig)
	mux.HandleFunc(pat.Get("/:type/"), ListBlobs)
	mux.HandleFunc(pat.Get("/:repo/:type/"), ListBlobs)
	mux.HandleFunc(pat.Head("/:type/:name"), CheckBlob)
	mux.HandleFunc(pat.Head("/:repo/:type/:name"), CheckBlob)
	mux.HandleFunc(pat.Get("/:type/:name"), GetBlob)
	mux.HandleFunc(pat.Get("/:repo/:type/:name"), GetBlob)
	mux.HandleFunc(pat.Post("/:type/:name"), SaveBlob)
	mux.HandleFunc(pat.Post("/:repo/:type/:name"), SaveBlob)
	mux.HandleFunc(pat.Delete("/:type/:name"), DeleteBlob)
	mux.HandleFunc(pat.Delete("/:repo/:type/:name"), DeleteBlob)

	return mux
}

var version = "manually"

func runRoot(cmd *cobra.Command, args []string) error {
	log.SetFlags(0)

	log.Printf("rest-server %s compiled with %v on %v/%v\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Printf("Data directory: %s", config.path)

	if config.cpuprofile != "" {
		f, err := os.Create(config.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		log.Println("CPU profiling enabled")
		defer pprof.StopCPUProfile()
	}

	mux := setupMux()

	var handler http.Handler
	htpasswdFile, err := NewHtpasswdFromFile(filepath.Join(config.path, ".htpasswd"))
	if err != nil {
		handler = mux
		log.Println("Authentication disabled")
	} else {
		handler = AuthHandler(htpasswdFile, mux)
		log.Println("Authentication enabled")
	}

	if !config.tls {
		log.Printf("Starting server on %s\n", config.listen)
		err = http.ListenAndServe(config.listen, handler)
	} else {
		privateKey := filepath.Join(config.path, "private_key")
		publicKey := filepath.Join(config.path, "public_key")
		log.Println("TLS enabled")
		log.Printf("Private key: %s", privateKey)
		log.Printf("Public key: %s", publicKey)
		log.Printf("Starting server on %s\n", config.listen)
		err = http.ListenAndServeTLS(config.listen, publicKey, privateKey, handler)
	}
	if err != nil {
		log.Fatal(err)
	}

	return nil

}

func main() {
	if err := cmdRoot.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
