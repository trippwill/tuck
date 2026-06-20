package status

import (
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pkgref"
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
	SourceID string
	Context  string
}

type Entry struct {
	TargetPath     string              `json:"targetPath"`
	State          string              `json:"state"`
	Package        string              `json:"package,omitempty"`
	Entry          string              `json:"entry,omitempty"`
	Code           target.ConflictCode `json:"code,omitempty"`
	Message        string              `json:"message,omitempty"`
	Owner          string              `json:"owner,omitempty"`
	ExpectedTarget string              `json:"expectedTarget,omitempty"`
}

type Result struct {
	Command string  `json:"-"`
	Context string  `json:"-"`
	Source  string  `json:"source"`
	Entries []Entry `json:"entries"`
}

func File(path string, options Options) (Result, error) {
	source, scope, err := active(options)
	if err != nil {
		return Result{}, err
	}
	targetPath, err := expandPath(path)
	if err != nil {
		return Result{}, err
	}
	class := target.ClassifyAt(targetPath, scope.PhysicalPath(targetPath), source, scope.Context, scope.LogicalRoot, nil, "")
	return Result{
		Command: "status",
		Context: scope.Context,
		Source:  source.ID,
		Entries: []Entry{entryFromClass(targetPath, "", "", class, false)},
	}, nil
}

func Package(ref string, options Options) (Result, error) {
	source, scope, err := active(options)
	if err != nil {
		return Result{}, err
	}
	var resolved []packages.Resolved
	if ref == "" {
		resolved, err = packages.Resolve(source, scope.Context, nil, true)
	} else {
		parsed, parseErr := pkgref.Parse(ref)
		if parseErr != nil {
			return Result{}, parseErr
		}
		var one packages.Resolved
		one, err = packages.ResolveOne(source, scope.Context, parsed.Name)
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
			targetEntry, err := target.NewPackageEntry(pkg, pkgEntry, scope.LogicalRoot, scope.PhysicalPath)
			if err != nil {
				entries = append(entries, Entry{
					TargetPath: targetEntry.TargetPath,
					State:      StateConflict,
					Package:    targetEntry.PackageID,
					Entry:      pkgEntry.Path,
					Code:       target.ConflictPathMismatch,
					Message:    err.Error(),
				})
				continue
			}
			class := targetEntry.Classify(source, scope.Context, scope.LogicalRoot)
			entries = append(entries, entryFromClass(targetEntry.TargetPath, targetEntry.PackageID, targetEntry.Entry.Path, class, true))
		}
	}

	return Result{
		Command: "package status",
		Context: scope.Context,
		Source:  source.ID,
		Entries: entries,
	}, nil
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
