package packages

import (
	"io/fs"
	"path/filepath"
	"sort"

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
	dirSet := make(map[string]struct{})
	for _, entry := range entries {
		if entry.Dir {
			dirSet[entry.Rel] = struct{}{}
		}
	}

	hasChildDir := make(map[string]struct{}, len(dirSet))
	for rel := range dirSet {
		for parent := filepath.Dir(rel); parent != "." && parent != rel; {
			if _, ok := dirSet[parent]; ok {
				hasChildDir[parent] = struct{}{}
			}
			next := filepath.Dir(parent)
			if next == parent {
				break
			}
			parent = next
		}
	}

	dirs := make([]Entry, 0)
	for _, entry := range entries {
		if !entry.Dir {
			continue
		}
		if _, ok := hasChildDir[entry.Rel]; ok {
			continue
		}
		dirs = append(dirs, entry)
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
