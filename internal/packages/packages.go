package packages

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

//go:generate go run ../../cmd/errgen -types ErrPackage
type ErrPackage string

const ErrPackageNotFound ErrPackage = "package not found"

const ContextHome = "home"

type Identity struct {
	Source  string
	Context string
	Name    string
	Root    string
}

func (p Identity) String() string {
	return p.Source + ":" + p.Context + ":" + p.Name
}

type Entry struct {
	Path string
	Rel  string
	Dir  bool
}

type Resolved struct {
	Identity Identity
	Entries  []Entry
}

type ListOptions struct {
	SourceID string
	Context  string
}

type Listing struct {
	Source   string
	Context  string
	Packages []string
}

func List(options ListOptions) (Listing, error) {
	context := options.Context
	if context == "" {
		context = ContextHome
	}
	registry, err := state.Load()
	if err != nil {
		return Listing{}, err
	}
	source, err := resolve.ActiveSource(registry, options.SourceID)
	if err != nil {
		return Listing{}, err
	}
	names, err := Discover(Base(source, context))
	if err != nil {
		return Listing{}, err
	}
	return Listing{
		Source:   source.ID,
		Context:  context,
		Packages: names,
	}, nil
}

func Base(source state.Source, context string) string {
	if context == "root" {
		return filepath.Join(source.Path, ".root")
	}
	return source.Path
}

func Resolve(source state.Source, context string, refs []string, all bool) ([]Resolved, error) {
	base := Base(source, context)
	if all {
		names, err := Discover(base)
		if err != nil {
			return nil, err
		}
		refs = names
	}

	seen := make(map[string]struct{}, len(refs))
	resolved := make([]Resolved, 0, len(refs))
	for _, raw := range refs {
		ref, err := pkgref.Parse(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[ref.Name]; ok {
			continue
		}
		seen[ref.Name] = struct{}{}

		pkg, err := ResolveOne(source, context, ref.Name)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, pkg)
	}
	return resolved, nil
}

func ResolveOne(source state.Source, context string, name string) (Resolved, error) {
	root := filepath.Join(Base(source, context), name)
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Resolved{}, AppErrMsgf(ErrPackageNotFound, "package %q not found in source %q", name, source.ID)
		}
		return Resolved{}, AppErrWrapf(ErrPackageNotFound, err, "could not inspect package %q in source %q", name, source.ID)
	}
	if !info.IsDir() {
		return Resolved{}, AppErrMsgf(ErrPackageNotFound, "package %q not found in source %q", name, source.ID)
	}
	entries, err := Enumerate(root)
	if err != nil {
		return Resolved{}, err
	}
	return Resolved{
		Identity: Identity{Source: source.ID, Context: context, Name: name, Root: root},
		Entries:  entries,
	}, nil
}

func Discover(base string) ([]string, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

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
