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
		LogVerbosef("%s, '%s'\n", typ, change.GetPath())
	}
}

/* for a new directory, git status returns:
?? js/
*/
func gitStatusShouldExpandDir(c *GitChange) bool {
	return c.Type == NotCheckedIn && strings.HasSuffix(c.GetPath(), "/")
}

func gitStatusExpandDirs(changes []*GitChange) []*GitChange {
	var res []*GitChange
	for _, c := range changes {
		if !gitStatusShouldExpandDir(c) {
			res = append(res, c)
			continue
		}
		filepath.Walk(c.GetPath(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			gc := &GitChange{
				PathAfter: path,
				Type:      NotCheckedIn,
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

func dumpDirDiffs(dirDiffs []*DirDiffEntry) {
	for _, e := range dirDiffs {
		tp := gitTypeToString(e.Type)
		fmt.Printf("'%s', '%s', %s\n", e.PathBefore, e.PathAfter, tp)
	}
}

func main() {
	parseFlags()
	if flgDev {
		verboseLogging = true
	}
	args := flag.Args()
	if len(args) == 2 {
		dirBefore := args[0]
		dirAfter := args[1]
		LogVerbosef("comparing 2 directories: '%s' and '%s'\n", dirBefore, dirAfter)
		dirDiffs, err := dirDiff(dirBefore, dirAfter)
		if err != nil {
			LogErrorf("dirDiff() failed with '%s'\n", err)
			os.Exit(1)
		}
		dumpDirDiffs(dirDiffs)
		os.Exit(0)
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
