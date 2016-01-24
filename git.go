package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	Modified = iota
	Added
	Deleted
	Renamed
	NotCheckedIn
)

type GitChange struct {
	Type  int // Modified, Added etc.
	Path  string
	Path2 string // only for Renamed
}

func (c *GitChange) GetName() string {
	return filepath.Base(c.Path)
}

var (
	gitPath      string
	gitTypeNames = []string{"Modified", "Added", "Deleted", "NotCheckedIn"}
)

func gitTypeToString(n int) string {
	return gitTypeNames[n]
}

func catGitHeadToFileMust(dst, gitPath string) {
	fmt.Printf("catGitHeadToFileMust: %s => %s\n", gitPath, dst)
	d := gitGetFileContentHeadMust(gitPath)
	f, err := os.Create(dst)
	fataliferr(err)
	defer f.Close()
	_, err = f.Write(d)
	fataliferr(err)
}

func parseGitStatusLineMust(s string) *GitChange {
	c := &GitChange{}
	parts := strings.SplitN(s, " ", 2)
	fatalif(len(parts) != 2, "invalid line: '%s'\n", s)
	switch parts[0] {
	case "M":
		c.Type = Modified
	case "A":
		c.Type = Added
	case "D":
		c.Type = Deleted
	case "??":
		c.Type = NotCheckedIn
	case "R":
		c.Type = Renamed
		// www/static/js/file_diff.js -> js/file_diff.js
		parts = strings.SplitN(parts[1], " -> ", 2)
		fatalif(len(parts) != 2, "invalid line: '%s'\n", s)
		c.Path = strings.TrimSpace(parts[0])
		c.Path2 = strings.TrimSpace(parts[1])
		return c
	default:
		fatalif(true, "invalid line: '%s'\n", s)
	}
	c.Path = strings.TrimSpace(parts[1])
	return c
}

func parseGitStatusMust(out []byte, includeNotCheckedIn bool) []*GitChange {
	var res []*GitChange
	lines := toTrimmedLines(out)
	for _, l := range lines {
		c := parseGitStatusLineMust(l)
		if !includeNotCheckedIn && c.Type == NotCheckedIn {
			continue
		}
		res = append(res, c)
	}
	return res
}

func gitStatusMust() []*GitChange {
	out, err := runCmd(gitPath, "status", "--porcelain")
	fataliferr(err)
	return parseGitStatusMust(out, true)
}

func gitGetFileContentHeadMust(path string) []byte {
	loc := "HEAD:" + path
	out, err := runCmd(gitPath, "show", loc)
	fataliferr(err)
	return out
}

func hasGitDirMust(dir string) bool {
	files, err := ioutil.ReadDir(dir)
	fataliferr(err)
	for _, fi := range files {
		if strings.ToLower(fi.Name()) == ".git" {
			return fi.IsDir()
		}
	}
	return false
}

// git status returns names relative to root of
func cdToGitRoot() {
	var newDir string
	dir, err := os.Getwd()
	fataliferr(err)
	for {
		if hasGitDirMust(dir) {
			break
		}
		newDir = filepath.Dir(dir)
		fatalif(dir == newDir, "dir == newDir (%s == %s)", dir, newDir)
		dir = newDir
	}
	if newDir != "" {
		fmt.Printf("Changed current dir to: '%s'\n", newDir)
		os.Chdir(newDir)
	}
}

func detectGitExeMust() {
	gitPath = detectExeMust("git")
}
