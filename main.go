package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	flgDev bool
)

// Change combines a GitChange and corresponding server response
type Change struct {
	GitChange
	ThickResponse
}

func dumpGitChanges(gitChanges []*GitChange) {
	for _, change := range gitChanges {
		typ := gitTypeToString(change.Type)
		LogVerbosef("%s, '%s'\n", typ, change.Path)
	}
}

/* for a new directory, git status returns:
?? js/
*/
func gitStatusShouldExpandDir(c *GitChange) bool {
	return c.Type == NotCheckedIn && strings.HasSuffix(c.Path, "/")
}

func gitStatusExpandDirs(changes []*GitChange) []*GitChange {
	var res []*GitChange
	for _, c := range changes {
		if !gitStatusShouldExpandDir(c) {
			res = append(res, c)
			continue
		}
		filepath.Walk(c.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			gc := &GitChange{
				Path: path,
				Type: NotCheckedIn,
			}
			res = append(res, gc)
			return nil
		})
	}
	return res
}

func parseFlags() {
	flag.BoolVar(&flgDev, "dev", false, "running in dev mode")
	flag.Parse()
}

func main() {
	parseFlags()
	if flgDev {
		verboseLogging = true
	}
	LogVerbosef("getting list of changed files\n")
	detectGitExeMust()
	cdToGitRoot()
	if hasZipResources() {
		LogVerbosef("Using resources from zip file\n")
		loadResourcesFromEmbeddedZip()
	}

	gitChanges := gitStatusMust()
	gitChanges = gitStatusExpandDirs(gitChanges)
	buildGlobalChanges(gitChanges)
	dumpGitChanges(gitChanges)
	if len(globalChanges) == 0 {
		fmt.Printf("There are no changes!\n")
		os.Exit(0)
	}
	startWebServer()
}
