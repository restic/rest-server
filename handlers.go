package main

import (
	"fmt"
	"net/http"
)

type Handler func(w http.ResponseWriter, r *http.Request, c *Context)

func HeadConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "head config")
}

func GetConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "get config")
}

func PostConfig(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "post config")
}

func ListBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "list blob")
}

func HeadBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "head blob")
}

func GetBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "get blob")
}

func PostBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "post blob")
}

func DeleteBlob(w http.ResponseWriter, r *http.Request, c *Context) {
	fmt.Fprintln(w, "delete blob")
}
