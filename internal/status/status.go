package status

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
	"github.com/trippwill/tuck/internal/target"
)

const (
	StateDeployed     = "deployed"
	StateAbsent       = "absent"
	StateConflict     = "conflict"
	StateMismatch     = "mismatch"
	StateOwnedByOther = "owned_by_other"
	StateUnmanaged    = "unmanaged"
)

type Options struct {
	SourceID   string
	TargetRoot string
}

type Entry struct {
	TargetPath     string `json:"targetPath"`
	State          string `json:"state"`
	Package        string `json:"package,omitempty"`
	Entry          string `json:"entry,omitempty"`
	Code           string `json:"code,omitempty"`
	Message        string `json:"message,omitempty"`
	Owner          string `json:"owner,omitempty"`
	ExpectedTarget string `json:"expectedTarget,omitempty"`
}

type Result struct {
	Command string  `json:"-"`
	Context string  `json:"-"`
	Source  string  `json:"source"`
	Entries []Entry `json:"entries"`
}

func File(path string, options Options) (Result, error) {
	source, targetRoot, err := active(options)
	if err != nil {
		return Result{}, err
	}
	targetPath, err := expandPath(path)
	if err != nil {
		return Result{}, err
	}
	class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, nil, "")
	return Result{
		Command: "status",
		Context: packages.ContextHome,
		Source:  source.ID,
		Entries: []Entry{entryFromClass(targetPath, "", "", class, false)},
	}, nil
}

func Package(ref string, options Options) (Result, error) {
	source, targetRoot, err := active(options)
	if err != nil {
		return Result{}, err
	}
	var resolved []packages.Resolved
	if ref == "" {
		resolved, err = packages.Resolve(source, packages.ContextHome, nil, true)
	} else {
		parsed, parseErr := pkgref.Parse(ref)
		if parseErr != nil {
			return Result{}, parseErr
		}
		var one packages.Resolved
		one, err = packages.ResolveOne(source, packages.ContextHome, parsed.Name)
		if err == nil {
			resolved = []packages.Resolved{one}
		}
	}
	if err != nil {
		return Result{}, err
	}

	entries := make([]Entry, 0)
	for _, pkg := range resolved {
		for _, pkgEntry := range packages.Leaves(pkg.Entries) {
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, pkgEntry.Path, targetRoot)
			if err != nil {
				entries = append(entries, Entry{
					TargetPath: targetPath,
					State:      StateConflict,
					Package:    pkg.Identity.String(),
					Entry:      pkgEntry.Path,
					Code:       "path_mismatch",
					Message:    err.Error(),
				})
				continue
			}
			class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, &pkg.Identity, pkgEntry.Rel)
			entries = append(entries, entryFromClass(targetPath, pkg.Identity.String(), pkgEntry.Path, class, true))
		}
	}

	return Result{
		Command: "package status",
		Context: packages.ContextHome,
		Source:  source.ID,
		Entries: entries,
	}, nil
}

func active(options Options) (state.Source, string, error) {
	targetRoot := options.TargetRoot
	if targetRoot == "" {
		targetRoot = os.Getenv("HOME")
	}
	if targetRoot == "" {
		targetRoot = "."
	}
	targetRoot = filepath.Clean(targetRoot)

	registry, err := state.Load()
	if err != nil {
		return state.Source{}, "", err
	}
	source, err := resolve.ActiveSource(registry, options.SourceID)
	if err != nil {
		return state.Source{}, "", err
	}
	return source, targetRoot, nil
}

func expandPath(raw string) (string, error) {
	path := raw
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = home
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = abs
	}
	return filepath.Clean(path), nil
}

func entryFromClass(targetPath string, selectedPackage string, selectedEntry string, class target.Class, selected bool) Entry {
	entry := Entry{
		TargetPath: targetPath,
		Package:    selectedPackage,
		Entry:      selectedEntry,
	}
	switch class.Kind {
	case target.Absent:
		entry.State = StateAbsent
	case target.Managed, target.ManagedSelected:
		entry.State = StateDeployed
	case target.ManagedOther:
		entry.State = StateOwnedByOther
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	case target.PathMismatch:
		entry.State = StateMismatch
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	case target.UnmanagedSymlink:
		entry.State = StateUnmanaged
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	default:
		entry.State = StateConflict
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	}

	if class.Owner.Identity.Name != "" {
		entry.Owner = class.Owner.Identity.String()
		if entry.Package == "" && !selected {
			entry.Package = class.Owner.Identity.String()
		}
		if entry.Entry == "" && !selected {
			entry.Entry = class.Owner.EntryPath
		}
		if entry.ExpectedTarget == "" && class.Owner.Mismatch {
			entry.ExpectedTarget = class.Owner.ExpectedTarget
		}
	}
	return entry
}
