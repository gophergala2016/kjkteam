package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/kjk/log"
	"github.com/kjk/u"
)

var (
	httpAddr      = "127.0.0.1:6111"
	mu            sync.Mutex
	globalChanges []*Change
)

const (
	TypeAdd    = "add"
	TypeDelete = "delete"
	TypeMove   = "move"
	TypeChange = "change"
)

// ThickResponse describes response for /thick/:idx
type ThickResponse struct {
	BeforePath *string `json:"a"`
	AfterPath  *string `json:"b"`
	IsImage    bool    `json:"is_image_diff"`
	NoChanges  bool    `json:"no_changes"`
	// Type is "add", "delete", "move", "change"
	Type          string `json:"type"`
	contentBefore []byte
	contentAfter  []byte
}

func gitChangeTypeToThickResponseType(typ int) string {
	switch typ {
	case Modified:
		return TypeChange
	case Added:
		return TypeAdd
	case Deleted:
		return TypeDelete
	case NotCheckedIn:
		return TypeAdd
	default:
		fatalf("unknown type: %d\n", typ)
		return "unknown"
	}
}

// ThickResponseFromGitChange creates ThickResponse out of GitChange
func ThickResponseFromGitChange(c *GitChange) ThickResponse {
	var res ThickResponse
	res.Type = gitChangeTypeToThickResponseType(c.Type)
	switch c.Type {
	case Modified:
		res.BeforePath = &c.Path
		res.AfterPath = &c.Path
		res.contentBefore = gitGetFileContentHeadMust(c.Path)
		res.contentAfter = readFileMust(c.Path)
	case Added:
		res.BeforePath = nil
		res.AfterPath = &c.Path
		res.contentBefore = nil
		res.contentAfter = readFileMust(c.Path)
	case Deleted:
		res.BeforePath = &c.Path
		res.AfterPath = nil
		res.contentBefore = gitGetFileContentHeadMust(c.Path)
		res.contentAfter = nil
	case NotCheckedIn:
		res.BeforePath = nil
		res.AfterPath = &c.Path
		res.contentBefore = nil
		res.contentAfter = readFileMust(c.Path)
	}
	res.IsImage = isImageFile(c.Path)
	res.NoChanges = bytes.Equal(res.contentBefore, res.contentAfter)
	return res
}

func buildGlobalChanges(gitChanges []*GitChange) {
	var changes []*Change
	for _, c := range gitChanges {
		gc := &Change{}
		gc.GitChange = *c
		gc.ThickResponse = ThickResponseFromGitChange(c)
		changes = append(changes, gc)
	}

	mu.Lock()
	globalChanges = changes
	mu.Unlock()
}

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

func acceptsGzip(r *http.Request) bool {
	return r != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func httpOkBytesWithContentType(w http.ResponseWriter, r *http.Request, contentType string, content []byte) {
	w.Header().Set("Content-Type", contentType)
	// https://www.maxcdn.com/blog/accept-encoding-its-vary-important/
	// prevent caching non-gzipped version
	w.Header().Add("Vary", "Accept-Encoding")
	if acceptsGzip(r) {
		w.Header().Set("Content-Encoding", "gzip")
		// Maybe: if len(content) above certain size, write as we go (on the other
		// hand, if we keep uncompressed data in memory...)
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write(content)
		gz.Close()
		content = buf.Bytes()
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Write(content)
}

func httpOkWithJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		// should never happen
		log.Errorf("json.MarshalIndent() failed with %q\n", err)
	}
	httpOkBytesWithContentType(w, r, "application/json", b)
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

func getThickResponseByIdx(idx int) *ThickResponse {
	mu.Lock()
	defer mu.Unlock()
	if idx >= len(globalChanges) {
		return nil
	}
	gc := globalChanges[idx]
	return &gc.ThickResponse
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
	tr := getThickResponseByIdx(idx)
	if tr == nil {
		http.NotFound(w, r)
		return
	}

	httpOkWithJSON(w, r, tr)
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
