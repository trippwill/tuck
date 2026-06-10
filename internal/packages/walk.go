package packages

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/trippwill/tuck/internal/pathutil"
)

func Enumerate(root string) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := pathutil.RelInside(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, Entry{Path: path, Rel: rel, Dir: dirEntry.IsDir()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Rel < entries[j].Rel
	})
	return entries, nil
}

func Directories(entries []Entry) []Entry {
	hasChildDir := make(map[string]bool)
	for _, candidate := range entries {
		if !candidate.Dir {
			continue
		}
		for _, other := range entries {
			if other.Dir && other.Rel != candidate.Rel && strings.HasPrefix(other.Rel, candidate.Rel+string(filepath.Separator)) {
				hasChildDir[candidate.Rel] = true
				break
			}
		}
	}
	dirs := make([]Entry, 0)
	for _, entry := range entries {
		if entry.Dir && !hasChildDir[entry.Rel] {
			dirs = append(dirs, entry)
		}
	}
	return dirs
}

func Leaves(entries []Entry) []Entry {
	leaves := make([]Entry, 0)
	for _, entry := range entries {
		if !entry.Dir {
			leaves = append(leaves, entry)
		}
	}
	return leaves
}
