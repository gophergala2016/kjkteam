package main

import (
	"fmt"
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
