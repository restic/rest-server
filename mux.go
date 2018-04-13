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

func (c Config) debugHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s", r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
}

func (c Config) logHandler(next http.Handler) http.Handler {
	accessLog, err := os.OpenFile(c.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return handlers.CombinedLoggingHandler(accessLog, next)
}

// NewHandler returns the master HTTP multiplexer/router.
func NewHandler(config Config) *goji.Mux {
	mux := goji.NewMux()

	if config.Debug {
		mux.Use(config.debugHandler)
	}

	if config.Log != "" {
		mux.Use(config.logHandler)
	}

	if config.Prometheus {
		mux.Handle(pat.Get("/metrics"), promhttp.Handler())
	}

	mux.HandleFunc(pat.Head("/config"), config.CheckConfig)
	mux.HandleFunc(pat.Head("/:repo/config"), config.CheckConfig)
	mux.HandleFunc(pat.Get("/config"), config.GetConfig)
	mux.HandleFunc(pat.Get("/:repo/config"), config.GetConfig)
	mux.HandleFunc(pat.Post("/config"), config.SaveConfig)
	mux.HandleFunc(pat.Post("/:repo/config"), config.SaveConfig)
	mux.HandleFunc(pat.Delete("/config"), config.DeleteConfig)
	mux.HandleFunc(pat.Delete("/:repo/config"), config.DeleteConfig)
	mux.HandleFunc(pat.Get("/:type/"), config.ListBlobs)
	mux.HandleFunc(pat.Get("/:repo/:type/"), config.ListBlobs)
	mux.HandleFunc(pat.Head("/:type/:name"), config.CheckBlob)
	mux.HandleFunc(pat.Head("/:repo/:type/:name"), config.CheckBlob)
	mux.HandleFunc(pat.Get("/:type/:name"), config.GetBlob)
	mux.HandleFunc(pat.Get("/:repo/:type/:name"), config.GetBlob)
	mux.HandleFunc(pat.Post("/:type/:name"), config.SaveBlob)
	mux.HandleFunc(pat.Post("/:repo/:type/:name"), config.SaveBlob)
	mux.HandleFunc(pat.Delete("/:type/:name"), config.DeleteBlob)
	mux.HandleFunc(pat.Delete("/:repo/:type/:name"), config.DeleteBlob)
	mux.HandleFunc(pat.Post("/"), config.CreateRepo)
	mux.HandleFunc(pat.Post("/:repo"), config.CreateRepo)
	mux.HandleFunc(pat.Post("/:repo/"), config.CreateRepo)

	return mux
}
