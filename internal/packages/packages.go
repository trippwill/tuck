package packages

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/state"
)

type ErrPackage string

func (e ErrPackage) Error() string { return string(e) }

const ErrPackageNotFound ErrPackage = "package not found"

const (
	ContextHome = domain.ContextHome
	ContextRoot = domain.ContextRoot
)

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
	Path   string
	Rel    string
	Dir    bool
	Deploy Deploy
	Mode   string
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

type ShowOptions struct {
	SourceID string
	Context  string
	Ref      string
}

type Tree struct {
	Command string      `json:"-"`
	Context string      `json:"-"`
	Source  string      `json:"-"`
	Package TreePackage `json:"package"`
}

type TreePackage struct {
	Identity string      `json:"identity"`
	Root     string      `json:"root"`
	Entries  []TreeEntry `json:"entries"`
}

type TreeEntry struct {
	Rel  string `json:"rel"`
	Type string `json:"type"`
}

func List(options ListOptions) (Listing, error) {
	context := domain.NormalizeContext(options.Context)
	source, err := domain.ActiveSource(options.SourceID)
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

func Show(options ShowOptions) (Tree, error) {
	context := domain.NormalizeContext(options.Context)
	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return Tree{}, err
	}
	ref, err := pkgref.Parse(options.Ref)
	if err != nil {
		return Tree{}, err
	}
	resolved, err := ResolveOne(source, context, ref.Name)
	if err != nil {
		return Tree{}, err
	}
	entries := make([]TreeEntry, 0, len(resolved.Entries))
	for _, entry := range resolved.Entries {
		entryType := "leaf"
		if entry.Dir {
			entryType = "dir"
		}
		entries = append(entries, TreeEntry{Rel: entry.Rel, Type: entryType})
	}
	return Tree{
		Command: "package show",
		Context: context,
		Source:  source.ID,
		Package: TreePackage{
			Identity: resolved.Identity.String(),
			Root:     resolved.Identity.Root,
			Entries:  entries,
		},
	}, nil
}

func Base(source state.Source, context string) string {
	if context == ContextRoot {
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
			return Resolved{}, apperr.AppErrMsgf(ErrPackageNotFound, "package %q not found in source %q", name, source.ID)
		}
		return Resolved{}, apperr.AppErrWrapf(ErrPackageNotFound, err, "could not inspect package %q in source %q", name, source.ID)
	}
	if !info.IsDir() {
		return Resolved{}, apperr.AppErrMsgf(ErrPackageNotFound, "package %q not found in source %q", name, source.ID)
	}
	entries, err := Enumerate(root)
	if err != nil {
		return Resolved{}, err
	}
	config, err := LoadConfig(root, entries)
	if err != nil {
		return Resolved{}, err
	}
	entries = ApplyConfig(entries, config)
	return Resolved{
		Identity: Identity{Source: source.ID, Context: context, Name: name, Root: root},
		Entries:  entries,
	}, nil
}

func ResolveIdentity(source state.Source, context string, rawRef string) (Identity, error) {
	ref, err := pkgref.Parse(rawRef)
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Source:  source.ID,
		Context: context,
		Name:    ref.Name,
		Root:    filepath.Join(Base(source, context), ref.Name),
	}, nil
}

func Discover(base string) ([]string, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
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
