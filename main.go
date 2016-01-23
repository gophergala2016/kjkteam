package main

import (
	"fmt"
	"sync"
)

var (
	mu            sync.Mutex
	globalChanges []*Change
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

// ThickResponseFromGitChange creates ThickResponse out of GitChange
func ThickResponseFromGitChange(c *GitChange) ThickResponse {
	var res ThickResponse
	res.Type = gitChangeTypeToThickResponseType(c.Type)
	res.AfterPath = c.Path
	res.BeforePath = c.Path
	// TODO: detect based on file extension
	res.IsImage = false
	// TODO: true if both files are the same
	res.NoChanges = false
	return res
}

func buildGlobalChanges(gitChanges []*GitChange) {
	var changes []*Change
	for _, c := range gitChanges {
		gc := &Change{}
		gc.GitChange = *c
		gc.ThickResponse = ThickResponseFromGitChange(c)
		changes = append(changes, gc)
	}

	mu.Lock()
	globalChanges = changes
	mu.Unlock()
}

func main() {
	fmt.Printf("getting list of changed files\n")
	detectGitExeMust()
	cdToGitRoot()
	gitChanges := gitStatusMust()
	dumpGitChanges(gitChanges)
	startWebServer()
}
