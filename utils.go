package main

import (
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func printStack() {
	buf := make([]byte, 1024*164)
	n := runtime.Stack(buf, false)
	fmt.Printf("%s", buf[:n])
}

func fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	printStack()
	os.Exit(1)
}

func fataliferr(err error) {
	if err != nil {
		fatalf("%s\n", err.Error())
	}
}

func fatalif(cond bool, format string, args ...interface{}) {
	if cond {
		fatalf(format, args...)
	}
}
func isMac() bool {
	return runtime.GOOS == "darwin"
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func detectExeMust(name string) string {
	path, err := exec.LookPath(name)
	if err == nil {
		//fmt.Printf("'%s' is '%s'\n", name, path)
		return path
	}
	fmt.Printf("Couldn't find '%s'\n", name)
	fataliferr(err)
	return ""
}

func toTrimmedLines(d []byte) []string {
	lines := strings.Split(string(d), "\n")
	i := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		// remove empty lines
		if len(l) > 0 {
			lines[i] = l
			i++
		}
	}
	return lines[:i]
}

// https://github.com/adobe/webkit/blob/master/Source/WebCore/platform/network/MIMESniffing.cpp#L90
// TODO: could be optimized to be a []bool or a bitmap
var binaryFlags = []byte{
	1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

func isBinaryData(d []byte) bool {
	n := len(d)
	if n > 512 {
		n = 512
	}
	for i := 0; i < n; i++ {
		b := d[i]
		if binaryFlags[b] == 1 {
			return true
		}
	}
	return false
}

func readFileMust(path string) []byte {
	d, err := ioutil.ReadFile(path)
	fataliferr(err)
	return d
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	knownImageExtensions := []string{".png", ".jpg", ".jpeg", ".bmp", ".tiff"}
	for _, s := range knownImageExtensions {
		if s == ext {
			return true
		}
	}
	return false
}

func runCmd(exePath string, args ...string) ([]byte, error) {
	cmd := exec.Command(exePath, args...)
	fmt.Printf("running: %s %v\n", filepath.Base(exePath), args)
	return cmd.Output()
}

func runCmdNoWait(exePath string, args ...string) error {
	cmd := exec.Command(exePath, args...)
	fmt.Printf("running: %s %v\n", filepath.Base(exePath), args)
	return cmd.Start()
}

var extraMimeTypes = map[string]string{
	".icon": "image-x-icon",
	".ttf":  "application/x-font-ttf",
	".woff": "application/x-font-woff",
	".eot":  "application/vnd.ms-fontobject",
	".svg":  "image/svg+xml",
}

// MimeTypeByExtensionExt is like mime.TypeByExtension but supports more types
// and defaults to text/plain
func MimeTypeByExtensionExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	result := mime.TypeByExtension(ext)

	if result == "" {
		result = extraMimeTypes[ext]
	}

	if result == "" {
		result = "text/plain; charset=utf-8"
	}

	return result
}

func openDefaultBrowserMac(uri string) error {
	cmd := exec.Command("open", uri)
	return cmd.Run()
}

func openDefaultBrowserWin(uri string) error {
	runDll32 := filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "rundll32.exe")
	return exec.Command(runDll32, "url.dll,FileProtocolHandler", uri).Run()
}

func openDefaultBrowser(uri string) error {
	var err error
	if isMac() {
		err = openDefaultBrowserMac(uri)
	} else if isWindows() {
		err = openDefaultBrowserWin(uri)
	} else {
		err = fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return err
}
