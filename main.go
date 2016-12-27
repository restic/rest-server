package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"

	"goji.io"
	"goji.io/pat"
)

var (
	path       = flag.String("path", "/tmp/restic", "data directory")
	listen     = flag.String("listen", ":8000", "listen address")
	tls        = flag.Bool("tls", false, "turn on TLS support")
	cpuprofile = flag.String("cpuprofile", "", "write CPU profile to file")
	debug      = flag.Bool("debug", false, "output debug messages")
)

func debugHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s", r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
}

func setupMux() *goji.Mux {
	mux := goji.NewMux()

	if *debug {
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

func main() {
	log.SetFlags(0)

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
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
	htpasswdFile, err := NewHtpasswdFromFile(filepath.Join(*path, ".htpasswd"))
	if err != nil {
		handler = mux
		log.Println("Authentication disabled")
	} else {
		handler = AuthHandler(htpasswdFile, mux)
		log.Println("Authentication enabled")
	}

	if !*tls {
		log.Printf("Starting server on %s\n", *listen)
		err = http.ListenAndServe(*listen, handler)
	} else {
		privateKey := filepath.Join(*path, "private_key")
		publicKey := filepath.Join(*path, "public_key")
		log.Println("TLS enabled")
		log.Printf("Private key: %s", privateKey)
		log.Printf("Public key: %s", publicKey)
		log.Printf("Starting server on %s\n", *listen)
		err = http.ListenAndServeTLS(*listen, publicKey, privateKey, handler)
	}
	if err != nil {
		log.Fatal(err)
	}
}
