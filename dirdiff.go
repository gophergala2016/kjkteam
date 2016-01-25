package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// DirDiffEntry describes a single file difference between
// files in two directories
type DirDiffEntry struct {
	PathBefore string
	PathAfter  string
	Type       int
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.IsDir()
}

// FileInfo describes a file
type FileInfo struct {
	Path string
	Size int64
}

func getFilesRecur(dir string) ([]FileInfo, error) {
	var res []FileInfo
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if strings.HasPrefix(path, dir) {
			path = path[len(dir):]
			if strings.HasPrefix(path, "/") {
				path = path[1:]
			}
		}
		fi := FileInfo{
			Path: path,
			Size: info.Size(),
		}
		res = append(res, fi)
		return nil
	})

	return res, err
}

// returnn true if files are equal
func filesEqual(path1, path2 string) (bool, error) {
	f1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer f1.Close()
	f2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer f2.Close()
	var buf1 [4096]byte
	var buf2 [4096]byte
	for {
		n1, err1 := f1.Read(buf1[:])
		if err1 != nil && err1 != io.EOF {
			return false, err1
		}
		n2, err2 := f2.Read(buf2[:])
		if err2 != nil && err2 != io.EOF {
			return false, err2
		}
		if n1 != n2 {
			return false, nil
		}
		if !bytes.Equal(buf1[:n1], buf2[:n1]) {
			return false, nil
		}
		if err1 != err2 {
			return false, nil
		}
		if err1 == io.EOF {
			return true, nil
		}
	}
}

func fileInfosToMap(fileInfos []FileInfo) map[string]int64 {
	res := make(map[string]int64)
	for _, fi := range fileInfos {
		res[fi.Path] = fi.Size
	}
	return res
}

func calcDirDiffs(rootBefore, rootAfter string, filesBefore, filesAfter map[string]int64) ([]*DirDiffEntry, error) {
	var res []*DirDiffEntry
	for pathBefore, sizeBefore := range filesBefore {
		fullPathBefore := filepath.Join(rootBefore, pathBefore)
		fullPathAfter := filepath.Join(rootAfter, pathBefore)
		var e DirDiffEntry
		sizeAfter, exists := filesAfter[pathBefore]
		if !exists {
			e.PathBefore = fullPathBefore
			e.Type = Deleted
			res = append(res, &e)
		} else {
			if sizeBefore != sizeAfter {
				e.PathBefore = fullPathBefore
				e.PathAfter = fullPathAfter
				e.Type = Modified
				res = append(res, &e)
			} else {
				areEqual, err := filesEqual(fullPathBefore, fullPathAfter)
				if err != nil {
					return nil, err
				}
				if !areEqual {
					e.PathBefore = fullPathBefore
					e.PathAfter = fullPathAfter
					res = append(res, &e)
				}
			}
		}
	}

	var pathsAfter []string
	for pathAfter := range filesAfter {
		if _, exists := filesBefore[pathAfter]; !exists {
			pathsAfter = append(pathsAfter, pathAfter)
		}
	}

	for _, pathAfter := range pathsAfter {
		fullPathAfter := filepath.Join(rootAfter, pathAfter)
		e := DirDiffEntry{
			PathBefore: "",
			PathAfter:  fullPathAfter,
			Type:       Added,
		}
		res = append(res, &e)
	}

	return res, nil
}

func dirDiff(pathBefore, pathAfter string) ([]*DirDiffEntry, error) {
	filesBefore, err := getFilesRecur(pathBefore)
	if err != nil {
		return nil, err
	}
	filesAfter, err := getFilesRecur(pathAfter)
	if err != nil {
		return nil, err
	}

	filesBeforeMap := fileInfosToMap(filesBefore)
	filesAfterMap := fileInfosToMap(filesAfter)

	return calcDirDiffs(pathBefore, pathAfter, filesBeforeMap, filesAfterMap)
}
