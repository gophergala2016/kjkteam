package main

import (
	"fmt"
	"os"
)

// Change combines a GitChange and corresponding server response
type Change struct {
	GitChange
	ThickResponse
}

func dumpGitChanges(gitChanges []*GitChange) {
	for _, change := range gitChanges {
		typ := gitTypeToString(change.Type)
		fmt.Printf("%s, '%s', '%s'\n", typ, change.Path, change.Name)
	}
}

func main() {
	fmt.Printf("getting list of changed files\n")
	detectGitExeMust()
	cdToGitRoot()
	gitChanges := gitStatusMust()
	buildGlobalChanges(gitChanges)
	dumpGitChanges(gitChanges)
	if len(globalChanges) == 0 {
		fmt.Printf("There are no changes!\n")
		os.Exit(0)
	}
	startWebServer()
}
