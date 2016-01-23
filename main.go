package main

import "fmt"

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
	dumpGitChanges(gitChanges)
}
