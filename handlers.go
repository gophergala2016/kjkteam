package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/kjk/u"
)

var (
	httpAddr = "127.0.0.1:6111"
)

func serveFile(w http.ResponseWriter, r *http.Request, fileName string) {
	//fmt.Printf("serverFile: fileName='%s'\n", fileName)
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
	fmt.Printf("%s '%s'\n", method, uri)
	path := uri[1:]
	if path == "" {
		path = "index.html"
	}
	serveFile(w, r, path)
}

// ThickResponse describes response for /thick/:idx
type ThickResponse struct {
	LeftPath  string `json:"a"`
	RightPath string `json:"b"`
	IsImage   bool   `json:"is_image_diff"`
	NoChanges bool   `json:"no_changes"`
	Type      string `json:"type"`
}

// /thick/:idx
func handleThick(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Path
	fmt.Printf("handleThick uri='%s'\n", uri)
	idxStr := uri[len("/thick/"):]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		fmt.Printf("missing argument in '%s'\n", uri)
		http.NotFound(w, r)
		return
	}
	fmt.Printf("idx: %d\n", idx)
	http.NotFound(w, r)
}

func registerHandlers() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/thick/", handleThick)
}

func startWebServer() {
	registerHandlers()
	fmt.Printf("Started runing on %s\n", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		fmt.Printf("http.ListendAndServer() failed with %s\n", err)
	}
	fmt.Printf("Exited\n")
}
