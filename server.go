package main

import (
	"io/ioutil"
	"net/http"

	"github.com/bchapuis/restic-server/config"
	"github.com/bchapuis/restic-server/handlers"
	"github.com/bchapuis/restic-server/router"
)

func main() {

	path, _ := ioutil.TempDir("", "restic-repository-")

	config.Init(path)

	r := router.NewRouter()

	r.FilterFunc(handlers.RequestLogger)

	r.HandleFunc("HEAD", "/config", handlers.HeadConfig)
	r.HandleFunc("GET", "/config", handlers.GetConfig)
	r.HandleFunc("POST", "/config", handlers.PostConfig)

	r.HandleFunc("HEAD", "/data", handlers.HeadData)
	r.HandleFunc("GET", "/data", handlers.GetData)
	r.HandleFunc("POST", "/data", handlers.PostData)
	r.HandleFunc("DELETE", "/data", handlers.DeleteData)

	r.HandleFunc("HEAD", "/snapshot", handlers.HeadSnapshot)
	r.HandleFunc("GET", "/snapshot", handlers.GetSnapshot)
	r.HandleFunc("POST", "/snapshot", handlers.PostSnapshot)
	r.HandleFunc("DELETE", "/snapshot", handlers.DeleteSnapshot)

	r.HandleFunc("HEAD", "/index", handlers.HeadIndex)
	r.HandleFunc("GET", "/index", handlers.GetIndex)
	r.HandleFunc("POST", "/index", handlers.PostIndex)
	r.HandleFunc("DELETE", "/index", handlers.DeleteIndex)

	r.HandleFunc("HEAD", "/lock", handlers.HeadLock)
	r.HandleFunc("GET", "/lock", handlers.GetLock)
	r.HandleFunc("POST", "/lock", handlers.PostLock)
	r.HandleFunc("DELETE", "/lock", handlers.DeleteLock)

	r.HandleFunc("HEAD", "/key", handlers.HeadKey)
	r.HandleFunc("GET", "/key", handlers.GetKey)
	r.HandleFunc("POST", "/key", handlers.PostKey)
	r.HandleFunc("DELETE", "/key", handlers.DeleteKey)

	http.ListenAndServe(":8000", r)
}
