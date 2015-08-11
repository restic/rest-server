package router

import (
	"net/http"
	"strings"
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
