package main

import "fmt"

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

	startWebServer()
}
