package main

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/kjk/u"
)

var (
	httpAddr = "127.0.0.1:6111"
)

func serveFile(w http.ResponseWriter, r *http.Request, fileName string) {
	fmt.Printf("serverFile: fileName='%s'\n", fileName)
	path := filepath.Join("www", fileName)
	if u.PathExists(path) {
		http.ServeFile(w, r, path)
	} else {
		fmt.Printf("file '%s' doesn't exist\n", path)
		http.NotFound(w, r)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Path
	method := r.Method
	fmt.Printf("uri: %s '%s'\n", method, uri)
	path := uri[1:]
	if path == "" {
		path = "index.html"
	}
	serveFile(w, r, path)
}

func registerHandlers() {
	http.HandleFunc("/", handleIndex)
}

func startWebServer() {
	registerHandlers()
	fmt.Printf("Started runing on %s\n", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		fmt.Printf("http.ListendAndServer() failed with %s\n", err)
	}
	fmt.Printf("Exited\n")
}
