package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

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
	Index         int    `json:"idx"`
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

const (
	// Megabyte is 1024 kilo-bytes
	Megabyte = 1024 * 1024
)

func capFileSize(d []byte) []byte {
	n := len(d)
	if n > Megabyte {
		s := fmt.Sprintf("Large file, size: %d bytes", n)
		return []byte(s)
	}
	if http.DetectContentType(d) == "application/octet-stream" {
		s := fmt.Sprintf("Binary file, size: %d bytes", n)
		return []byte(s)
	}
	return d
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
	res.contentBefore = capFileSize(res.contentBefore)
	res.contentAfter = capFileSize(res.contentAfter)
	res.IsImage = isImageFile(c.Path)
	res.NoChanges = bytes.Equal(res.contentBefore, res.contentAfter)
	return res
}

func buildGlobalChanges(gitChanges []*GitChange) {
	var changes []*Change
	for i, c := range gitChanges {
		gc := &Change{}
		gc.GitChange = *c
		gc.ThickResponse = ThickResponseFromGitChange(c)
		gc.ThickResponse.Index = i
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

func serveIndexPage(w http.ResponseWriter, r *http.Request) {
	var pairs []*ThickResponse
	mu.Lock()
	for _, gc := range globalChanges {
		pairs = append(pairs, &gc.ThickResponse)
	}
	mu.Unlock()
	v := struct {
		Pairs []*ThickResponse
	}{
		Pairs: pairs,
	}
	execTemplate(w, tmplIndex, v)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Path
	method := r.Method
	fmt.Printf("%s '%s'\n", method, uri)
	path := uri[1:]
	if path == "" {
		serveIndexPage(w, r)
		return
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

func strPtrToLower(s *string) string {
	if s == nil {
		return ""
	}
	return strings.ToLower(*s)
}

func findByPath(path string) *ThickResponse {
	path = strings.ToLower(path)
	mu.Lock()
	defer mu.Unlock()
	var p string
	for _, gc := range globalChanges {
		p = strPtrToLower(gc.BeforePath)
		if p == path {
			return &gc.ThickResponse
		}
		p = strPtrToLower(gc.AfterPath)
		if p == path {
			return &gc.ThickResponse
		}
	}
	return nil
}

func handleGetContents(w http.ResponseWriter, r *http.Request, which string) {
	path := r.FormValue("path")
	fmt.Printf("/%s/get_contents, path='%s'\n", which, path)
	tr := findByPath(path)
	if tr == nil {
		http.NotFound(w, r)
		return
	}
	var d []byte
	if which == "a" {
		d = tr.contentBefore
	} else {
		d = tr.contentAfter
	}
	mime := MimeTypeByExtensionExt(path)
	httpOkBytesWithContentType(w, r, mime, d)
}

func handdleGetContentsA(w http.ResponseWriter, r *http.Request) {
	handleGetContents(w, r, "a")
}

func handdleGetContentsB(w http.ResponseWriter, r *http.Request) {
	handleGetContents(w, r, "b")
}

func handleKill(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("handleKill, url: '%s'\n", r.URL.Path)
	os.Exit(0)
}

func registerHandlers() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/thick/", handleThick)
	http.HandleFunc("/a/get_contents", handdleGetContentsA)
	http.HandleFunc("/b/get_contents", handdleGetContentsB)
	http.HandleFunc("/kill", handleKill)
}

func startWebServer() {
	registerHandlers()

	go func(uri string) {
		time.Sleep(time.Second)
		fmt.Printf("Opening browser with '%s'\n", uri)
		openDefaultBrowser(uri)
	}("http://" + httpAddr)

	fmt.Printf("Started runing on %s\n", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		fmt.Printf("http.ListendAndServer() failed with %s\n", err)
	}
	fmt.Printf("Exited\n")
}
