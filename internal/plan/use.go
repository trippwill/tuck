package plan

import (
	"errors"
	"fmt"
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

//go:generate go run ../../cmd/errgen -types ErrPlan
type ErrPlan string

const (
	ErrPackageNotFound ErrPlan = "package not found"
	ErrApply           ErrPlan = "plan apply failed"
)

const ContextHome = "home"

type UseOptions struct {
	Refs       []string
	All        bool
	SourceID   string
	TargetRoot string
	Apply      bool
}

type PackageIdentity struct {
	Source  string
	Context string
	Name    string
	Root    string
}

func (p PackageIdentity) String() string {
	return p.Source + ":" + p.Context + ":" + p.Name
}

type Action struct {
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	LinkPath string `json:"linkPath,omitempty"`
	Payload  string `json:"payload,omitempty"`
	Target   string `json:"target,omitempty"`
}

type Conflict struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Package string `json:"package,omitempty"`
}

type Privilege struct {
	Required bool `json:"required"`
}

type UsePlan struct {
	Command   string     `json:"-"`
	Context   string     `json:"-"`
	DryRun    bool       `json:"dryRun"`
	Applied   bool       `json:"applied"`
	Packages  []string   `json:"packages"`
	Privilege Privilege  `json:"privilege"`
	Actions   []Action   `json:"actions"`
	Conflicts []Conflict `json:"conflicts"`
}

type entry struct {
	path string
	rel  string
	dir  bool
}

type resolvedPackage struct {
	identity PackageIdentity
	entries  []entry
}

func BuildUse(options UseOptions) (UsePlan, error) {
	targetRoot := options.TargetRoot
	if targetRoot == "" {
		targetRoot = os.Getenv("HOME")
	}
	if targetRoot == "" {
		return UsePlan{}, AppErrMsg(ErrApply, "HOME is not set")
	}
	targetRoot = filepath.Clean(targetRoot)

	registry, err := state.Load()
	if err != nil {
		return UsePlan{}, err
	}
	source, err := resolve.ActiveSource(registry, options.SourceID)
	if err != nil {
		return UsePlan{}, err
	}

	packages, err := resolvePackages(source, options)
	if err != nil {
		return UsePlan{}, err
	}

	usePlan := UsePlan{
		Command:   "package use",
		Context:   ContextHome,
		DryRun:    !options.Apply,
		Applied:   false,
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}
	for _, pkg := range packages {
		usePlan.Packages = append(usePlan.Packages, pkg.identity.String())
	}

	plannedTargets := make(map[string]string)
	for _, pkg := range packages {
		blockedRelPrefixes := make([]string, 0)
		for _, entry := range packageDirectories(pkg.entries) {
			targetPath, _, err := pathutil.PackageToTarget(pkg.identity.Root, entry.path, targetRoot)
			if err != nil {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict("path_mismatch", targetPath, pkg.identity.String(), err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.rel)
				continue
			}
			class := classifyTarget(targetPath, source, targetRoot, &pkg.identity, entry.rel)
			switch class.kind {
			case targetAbsent:
				usePlan.Actions = append(usePlan.Actions, Action{Type: "mkdir", Path: targetPath})
			case targetRealDirectory:
			default:
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(class.conflictCode(), targetPath, pkg.identity.String(), class.message))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.rel)
			}
		}

		for _, entry := range packageLeaves(pkg.entries) {
			if blockedByDirectoryConflict(entry.rel, blockedRelPrefixes) {
				continue
			}
			targetPath, _, err := pathutil.PackageToTarget(pkg.identity.Root, entry.path, targetRoot)
			if err != nil {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict("path_mismatch", targetPath, pkg.identity.String(), err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetPath]; ok && owner != pkg.identity.String()+":"+entry.rel {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict("multiple_providers", targetPath, pkg.identity.String(), "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetPath] = pkg.identity.String() + ":" + entry.rel

			class := classifyTarget(targetPath, source, targetRoot, &pkg.identity, entry.rel)
			switch class.kind {
			case targetAbsent:
				payload, err := pathutil.SymlinkPayload(targetPath, entry.path)
				if err != nil {
					usePlan.Conflicts = append(usePlan.Conflicts, conflict("path_mismatch", targetPath, pkg.identity.String(), err.Error()))
					continue
				}
				usePlan.Actions = append(usePlan.Actions, Action{Type: "symlink", LinkPath: targetPath, Payload: payload, Target: entry.path})
			case targetManagedSelected:
			default:
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(class.conflictCode(), targetPath, pkg.identity.String(), class.message))
			}
		}
	}

	if len(usePlan.Conflicts) > 0 {
		usePlan.Actions = []Action{}
		return usePlan, nil
	}
	if options.Apply {
		if err := Apply(usePlan); err != nil {
			return usePlan, err
		}

		usePlan.Applied = true
		usePlan.DryRun = false
	}
	return usePlan, nil
}

func blockedByDirectoryConflict(rel string, blockedPrefixes []string) bool {
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func Apply(usePlan UsePlan) error {
	if len(usePlan.Conflicts) > 0 {
		return AppErrMsg(ErrApply, "cannot apply a plan with conflicts")
	}
	for _, action := range usePlan.Actions {
		switch action.Type {
		case "mkdir":
			if err := os.MkdirAll(action.Path, 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", action.Path)
			}
		case "symlink":
			if err := os.MkdirAll(filepath.Dir(action.LinkPath), 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", filepath.Dir(action.LinkPath))
			}
			if err := os.Symlink(action.Payload, action.LinkPath); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create symlink %q", action.LinkPath)
			}
		}
	}
	return nil
}

func resolvePackages(source state.Source, options UseOptions) ([]resolvedPackage, error) {
	if options.All {
		names, err := discoverPackages(source.Path)
		if err != nil {
			return nil, err
		}
		options.Refs = names
	}

	seen := make(map[string]struct{}, len(options.Refs))
	var packages []resolvedPackage
	for _, raw := range options.Refs {
		ref, err := pkgref.Parse(raw)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[ref.Name]; ok {
			continue
		}
		seen[ref.Name] = struct{}{}

		root := filepath.Join(source.Path, ref.Name)
		info, err := os.Stat(root)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, AppErrMsgf(ErrPackageNotFound, "package %q not found in source %q", ref.Name, source.ID)
			}
			return nil, AppErrWrapf(ErrPackageNotFound, err, "could not inspect package %q in source %q", ref.Name, source.ID)
		}
		if !info.IsDir() {
			return nil, AppErrMsgf(ErrPackageNotFound, "package %q not found in source %q", ref.Name, source.ID)
		}
		entries, err := enumerateEntries(root)
		if err != nil {
			return nil, err
		}
		packages = append(packages, resolvedPackage{
			identity: PackageIdentity{Source: source.ID, Context: ContextHome, Name: ref.Name, Root: root},
			entries:  entries,
		})
	}
	return packages, nil
}

func discoverPackages(base string) ([]string, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".root" {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func enumerateEntries(root string) ([]entry, error) {
	var entries []entry
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
		entries = append(entries, entry{path: path, rel: rel, dir: dirEntry.IsDir()})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})
	return entries, nil
}

func packageDirectories(entries []entry) []entry {
	hasChildDir := make(map[string]bool)
	for _, candidate := range entries {
		if !candidate.dir {
			continue
		}
		for _, other := range entries {
			if other.dir && other.rel != candidate.rel && strings.HasPrefix(other.rel, candidate.rel+string(filepath.Separator)) {
				hasChildDir[candidate.rel] = true
				break
			}
		}
	}
	var dirs []entry
	for _, entry := range entries {
		if entry.dir && !hasChildDir[entry.rel] {
			dirs = append(dirs, entry)
		}
	}
	return dirs
}

func packageLeaves(entries []entry) []entry {
	var leaves []entry
	for _, entry := range entries {
		if !entry.dir {
			leaves = append(leaves, entry)
		}
	}
	return leaves
}

type targetKind int

const (
	targetAbsent targetKind = iota
	targetRealDirectory
	targetRealFile
	targetSpecial
	targetUnmanagedSymlink
	targetManagedSelected
	targetManagedOther
	targetPathMismatch
)

type targetClass struct {
	kind    targetKind
	message string
}

func classifyTarget(targetPath string, source state.Source, targetRoot string, selected *PackageIdentity, selectedRel string) targetClass {
	info, err := os.Lstat(targetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return targetClass{kind: targetAbsent}
		}
		return targetClass{kind: targetSpecial, message: err.Error()}
	}
	if info.Mode()&os.ModeSymlink == 0 {
		if info.IsDir() {
			return targetClass{kind: targetRealDirectory, message: "target is a real directory"}
		}
		if info.Mode().IsRegular() {
			return targetClass{kind: targetRealFile, message: "target is a real file"}
		}
		return targetClass{kind: targetSpecial, message: "target is a special file"}
	}

	owner, ok := inferOwner(targetPath, source, targetRoot)
	if !ok {
		return targetClass{kind: targetUnmanagedSymlink, message: "target is an unmanaged symlink"}
	}
	if owner.mismatch {
		return targetClass{kind: targetPathMismatch, message: "managed symlink path does not match package entry"}
	}
	if selected != nil && owner.pkg == selected.Name && owner.rel == selectedRel {
		return targetClass{kind: targetManagedSelected}
	}
	return targetClass{kind: targetManagedOther, message: fmt.Sprintf("target is managed by %s:home:%s", source.ID, owner.pkg)}
}

func (c targetClass) conflictCode() string {
	switch c.kind {
	case targetRealDirectory:
		return "real_directory"
	case targetRealFile:
		return "real_file"
	case targetSpecial:
		return "special_file"
	case targetUnmanagedSymlink:
		return "unmanaged_symlink"
	case targetManagedOther:
		return "owned_by_other"
	case targetPathMismatch:
		return "path_mismatch"
	default:
		return "conflict"
	}
}

type owner struct {
	pkg      string
	rel      string
	mismatch bool
}

func inferOwner(linkPath string, source state.Source, targetRoot string) (owner, bool) {
	payload, err := os.Readlink(linkPath)
	if err != nil {
		return owner{}, false
	}
	targetAbs := payload
	if !filepath.IsAbs(payload) {
		targetAbs = filepath.Join(filepath.Dir(linkPath), payload)
	}
	targetAbs = filepath.Clean(targetAbs)
	base := filepath.Clean(source.Path)
	if !pathutil.Inside(targetAbs, base) {
		return owner{}, false
	}
	relToBase, err := pathutil.RelInside(base, targetAbs)
	if err != nil {
		return owner{}, false
	}
	parts := strings.Split(relToBase, string(filepath.Separator))
	if len(parts) < 2 {
		return owner{}, false
	}
	pkgName := parts[0]
	packageRoot := filepath.Join(base, pkgName)
	packageRel, err := pathutil.RelInside(packageRoot, targetAbs)
	if err != nil {
		return owner{}, false
	}
	expectedTarget := filepath.Clean(filepath.Join(targetRoot, packageRel))
	return owner{pkg: pkgName, rel: packageRel, mismatch: filepath.Clean(linkPath) != expectedTarget}, true
}

func conflict(code, path, pkg, message string) Conflict {
	return Conflict{Code: code, Path: path, Package: pkg, Message: message}
}
