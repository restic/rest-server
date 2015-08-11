package main

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/bchapuis/restic-server/config"
	"github.com/bchapuis/restic-server/handlers"
)

type Route struct {
	method  string
	pattern string
	handler http.Handler
}

type Router struct {
	filters []http.Handler
	routes  []Route
}

func NewRouter() Router {
	filters := []http.Handler{}
	routes := []Route{}
	return Router{filters, routes}
}

func (router *Router) Filter(handler http.Handler) {
	router.filters = append(router.filters, handler)
}

func (router *Router) FilterFunc(handlerFunc http.HandlerFunc) {
	router.Filter(handlerFunc)
}

func (router *Router) Handle(method string, pattern string, handler http.Handler) {
	router.routes = append(router.routes, Route{method, pattern, handler})
}

func (router *Router) HandleFunc(method string, pattern string, handlerFunc http.HandlerFunc) {
	router.Handle(method, pattern, handlerFunc)
}

func (router Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for i := 0; i < len(router.filters); i++ {
		filter := router.filters[i]
		filter.ServeHTTP(w, r)
	}
	for i := 0; i < len(router.routes); i++ {
		route := router.routes[i]
		if route.method == r.Method && strings.HasPrefix(r.URL.String(), route.pattern) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func main() {
	path, _ := ioutil.TempDir("", "restic-repository-")

	config.Init(path)

	router := NewRouter()

	router.FilterFunc(handlers.RequestLogger)

	router.HandleFunc("HEAD", "/config", handlers.HeadConfig)
	router.HandleFunc("GET", "/config", handlers.GetConfig)
	router.HandleFunc("POST", "/config", handlers.PostConfig)

	router.HandleFunc("HEAD", "/data", handlers.HeadData)
	router.HandleFunc("GET", "/data", handlers.GetData)
	router.HandleFunc("POST", "/data", handlers.PostData)
	router.HandleFunc("DELETE", "/data", handlers.DeleteData)

	router.HandleFunc("HEAD", "/snapshot", handlers.HeadSnapshot)
	router.HandleFunc("GET", "/snapshot", handlers.GetSnapshot)
	router.HandleFunc("POST", "/snapshot", handlers.PostSnapshot)
	router.HandleFunc("DELETE", "/snapshot", handlers.DeleteSnapshot)

	router.HandleFunc("HEAD", "/index", handlers.HeadIndex)
	router.HandleFunc("GET", "/index", handlers.GetIndex)
	router.HandleFunc("POST", "/index", handlers.PostIndex)
	router.HandleFunc("DELETE", "/index", handlers.DeleteIndex)

	router.HandleFunc("HEAD", "/lock", handlers.HeadLock)
	router.HandleFunc("GET", "/lock", handlers.GetLock)
	router.HandleFunc("POST", "/lock", handlers.PostLock)
	router.HandleFunc("DELETE", "/lock", handlers.DeleteLock)

	router.HandleFunc("HEAD", "/key", handlers.HeadKey)
	router.HandleFunc("GET", "/key", handlers.GetKey)
	router.HandleFunc("POST", "/key", handlers.PostKey)
	router.HandleFunc("DELETE", "/key", handlers.DeleteKey)

	http.ListenAndServe(":8000", router)
}
