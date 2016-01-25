package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

	// loaded only once at startup. maps a file path of the resource
	// to its data
	resourcesFromZip map[string][]byte
)

const (
	maxFileSizeToDiff = 1024 * 256 // 256 KB
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
	case Renamed:
		return TypeMove
	case NotCheckedIn:
		return TypeAdd
	default:
		fatalf("unknown type: %d\n", typ)
		return "unknown"
	}
}

func hasZipResources() bool {
	return len(resourcesZipData) > 0
}

func capFileSize(d []byte) []byte {
	if isBinaryData(d) || len(d) > maxFileSizeToDiff {
		var s string
		if isBinaryData(d) && len(d) > maxFileSizeToDiff {
			s = fmt.Sprintf("Not showing large (%d bytes), binary file. Size limit is %d bytes", len(d), maxFileSizeToDiff)
		} else {
			if isBinaryData(d) {
				s = fmt.Sprintf("Not showing binary file (%d bytes).", len(d))
			} else {
				s = fmt.Sprintf("Not showing large (%d bytes) file. Size limit is %d bytes", len(d), maxFileSizeToDiff)
			}
		}
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
		res.BeforePath = &c.PathBefore
		res.AfterPath = &c.PathBefore
		res.contentBefore = gitGetFileContentHeadMust(c.PathBefore)
		res.contentAfter = readFileMust(c.PathBefore)
	case Added:
		res.BeforePath = nil
		res.AfterPath = &c.PathAfter
		res.contentBefore = nil
		res.contentAfter = readFileMust(c.PathAfter)
	case Deleted:
		res.BeforePath = &c.PathBefore
		res.AfterPath = nil
		res.contentBefore = gitGetFileContentHeadMust(c.PathBefore)
		res.contentAfter = nil
	case Renamed:
		res.BeforePath = &c.PathBefore
		res.AfterPath = &c.PathAfter
		res.contentBefore = gitGetFileContentHeadMust(c.PathBefore)
		res.contentAfter = readFileMust(c.PathAfter)
	case NotCheckedIn:
		res.BeforePath = nil
		res.AfterPath = &c.PathAfter
		res.contentBefore = nil
		res.contentAfter = readFileMust(c.PathAfter)
	}
	res.contentBefore = capFileSize(res.contentBefore)
	res.contentAfter = capFileSize(res.contentAfter)
	res.IsImage = isImageFile(c.GetPath())
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

func normalizePath(s string) string {
	return strings.Replace(s, "\\", "/", -1)
}

func loadResourcesFromZipReader(zr *zip.Reader) error {
	for _, f := range zr.File {
		name := normalizePath(f.Name)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		d, err := ioutil.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		// for simplicity of the build, the file that we embedded in zip
		// is bundle.min.js but the html refers to it as bundle.js
		if name == "s/dist/bundle.min.js" {
			name = "s/dist/bundle.js"
		}
		//LogInfof("Loaded '%s' of size %d bytes\n", name, len(d))
		resourcesFromZip[name] = d
	}
	return nil
}

// call this only once at startup
func loadResourcesFromZip(path string) error {
	resourcesFromZip = make(map[string][]byte)
	zrc, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zrc.Close()
	return loadResourcesFromZipReader(&zrc.Reader)
}

func loadResourcesFromEmbeddedZip() error {
	//LogInfof("loadResourcesFromEmbeddedZip()\n")
	n := len(resourcesZipData)
	if n == 0 {
		return errors.New("len(resourcesZipData) == 0")
	}
	resourcesFromZip = make(map[string][]byte)
	r := bytes.NewReader(resourcesZipData)
	zrc, err := zip.NewReader(r, int64(n))
	if err != nil {
		return err
	}
	return loadResourcesFromZipReader(zrc)
}

func serveData(w http.ResponseWriter, r *http.Request, code int, contentType string, data, gzippedData []byte) {
	d := data
	if len(contentType) > 0 {
		w.Header().Set("Content-Type", contentType)
	}
	// https://www.maxcdn.com/blog/accept-encoding-its-vary-important/
	// prevent caching non-gzipped version
	w.Header().Add("Vary", "Accept-Encoding")

	if acceptsGzip(r) && len(gzippedData) > 0 {
		d = gzippedData
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(d)))
	w.WriteHeader(code)
	w.Write(d)
}

func serveResourceFromZip(w http.ResponseWriter, r *http.Request, path string) {
	path = normalizePath(path)
	LogVerbosef("serving '%s' from zip\n", path)

	data := resourcesFromZip[path]
	gzippedData := resourcesFromZip[path+".gz"]

	if data == nil {
		LogVerbosef("no data for file '%s'\n", path)
		servePlainText(w, r, 404, fmt.Sprintf("file '%s' not found", path))
		return
	}

	if len(data) == 0 {
		servePlainText(w, r, 404, "Asset is empty")
		return
	}

	serveData(w, r, 200, MimeTypeByExtensionExt(path), data, gzippedData)
}

func serveFile(w http.ResponseWriter, r *http.Request, fileName string) {
	//LogVerbosef("serverFile: fileName='%s'\n", fileName)
	path := filepath.Join("www", fileName)
	if hasZipResources() {
		serveResourceFromZip(w, r, path)
		return
	}
	if u.PathExists(path) {
		http.ServeFile(w, r, path)
	} else {
		LogVerbosef("file '%s' doesn't exist\n", path)
		http.NotFound(w, r)
	}
}

func acceptsGzip(r *http.Request) bool {
	return r != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func writeHeader(w http.ResponseWriter, code int, contentType string) {
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	w.WriteHeader(code)
}

func servePlainText(w http.ResponseWriter, r *http.Request, code int, format string, args ...interface{}) {
	writeHeader(w, code, "text/plain")
	var err error
	if len(args) > 0 {
		_, err = w.Write([]byte(fmt.Sprintf(format, args...)))
	} else {
		_, err = w.Write([]byte(format))
	}
	if err != nil {
		LogErrorf("err: '%s'\n", err)
	}
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
	LogVerbosef("%s '%s'\n", method, uri)
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
	LogVerbosef("handleThick uri='%s'\n", uri)
	idxStr := uri[len("/thick/"):]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		LogErrorf("missing argument in '%s'\n", uri)
		http.NotFound(w, r)
		return
	}
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
	LogVerbosef("/%s/get_contents, path='%s'\n", which, path)
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
	// application/json confuses front-end because jQuery ajax
	// automatically translate those to objects
	if mime == "application/json" || mime == "application/javascript" {
		mime = "text/plain"
	}
	httpOkBytesWithContentType(w, r, mime, d)
}

func handdleGetContentsA(w http.ResponseWriter, r *http.Request) {
	handleGetContents(w, r, "a")
}

func handdleGetContentsB(w http.ResponseWriter, r *http.Request) {
	handleGetContents(w, r, "b")
}

func handleKill(w http.ResponseWriter, r *http.Request) {
	LogVerbosef("handleKill, url: '%s'\n", r.URL.Path)
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
		LogVerbosef("Opening browser with '%s'\n", uri)
		openDefaultBrowser(uri)
	}("http://" + httpAddr)

	LogVerbosef("Started runing on %s\n", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		LogErrorf("http.ListendAndServer() failed with %s\n", err)
	}
	LogVerbosef("Exited\n")
}
