package restserver

import (
	"log"
	"net/http"
	"os"

	goji "goji.io"

	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"goji.io/pat"
)

// Config struct holds program configuration.
var Config = struct {
	Path       string
	Listen     string
	Log        string
	CPUProfile string
	TLS        bool
	TLSKey     string
	TLSCert    string
	AppendOnly bool
	Prometheus bool
	Debug      bool
}{
	Path:       "/tmp/restic",
	Listen:     ":8000",
	AppendOnly: false,
}

func debugHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s", r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
}

func logHandler(next http.Handler) http.Handler {
	accessLog, err := os.OpenFile(Config.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return handlers.CombinedLoggingHandler(accessLog, next)
}

// NewMux is master HTTP multiplexer/router.
func NewMux() *goji.Mux {
	mux := goji.NewMux()

	if Config.Debug {
		mux.Use(debugHandler)
	}

	if Config.Log != "" {
		mux.Use(logHandler)
	}

	if Config.Prometheus {
		mux.Handle(pat.Get("/metrics"), promhttp.Handler())
	}

	mux.HandleFunc(pat.Head("/config"), CheckConfig)
	mux.HandleFunc(pat.Head("/:repo/config"), CheckConfig)
	mux.HandleFunc(pat.Get("/config"), GetConfig)
	mux.HandleFunc(pat.Get("/:repo/config"), GetConfig)
	mux.HandleFunc(pat.Post("/config"), SaveConfig)
	mux.HandleFunc(pat.Post("/:repo/config"), SaveConfig)
	mux.HandleFunc(pat.Delete("/config"), DeleteConfig)
	mux.HandleFunc(pat.Delete("/:repo/config"), DeleteConfig)
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
	mux.HandleFunc(pat.Post("/"), CreateRepo)
	mux.HandleFunc(pat.Post("/:repo"), CreateRepo)
	mux.HandleFunc(pat.Post("/:repo/"), CreateRepo)

	return mux
}
