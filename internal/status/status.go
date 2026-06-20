package status

import (
	"path/filepath"

	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/pkgref"
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
	StateCopyMissing  = "copy_missing"
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
	source, scope, registry, err := active(options)
	if err != nil {
		return Result{}, err
	}
	targetPath, err := pathutil.ExpandInput(path)
	if err != nil {
		return Result{}, err
	}
	if !pathutil.Inside(targetPath, scope.LogicalRoot) {
		return Result{
			Command: "status",
			Context: scope.Context,
			Source:  source.ID,
			Entries: []Entry{{
				TargetPath: targetPath,
				State:      StateConflict,
				Code:       target.ConflictOutsideTargetRoot,
				Message:    "target is outside the selected target root; pass --root to use the root context",
			}},
		}, nil
	}
	if record, ok := registry.CopyByTarget(source.ID, scope.Context, targetPath); ok {
		copyEntry, err := packageEntryFromCopyRecord(source, scope, record)
		if err == nil {
			return Result{
				Command: "status",
				Context: scope.Context,
				Source:  source.ID,
				Entries: []Entry{entryFromCopyClass(copyEntry.TargetPath, copyEntry.PackageID, copyEntry.Entry.Path, target.ClassifyCopy(copyEntry, registry))},
			}, nil
		}
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
	source, scope, registry, err := active(options)
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
			if pkgEntry.Deploy == packages.DeployCopy {
				entries = append(entries, entryFromCopyClass(targetEntry.TargetPath, targetEntry.PackageID, targetEntry.Entry.Path, target.ClassifyCopy(targetEntry, registry)))
				continue
			}
			if _, ok := registry.CopyByEntry(pkg.Identity.Source, pkg.Identity.Context, pkg.Identity.Name, pkgEntry.Rel); ok {
				entries = append(entries, entryFromCopyClass(targetEntry.TargetPath, targetEntry.PackageID, targetEntry.Entry.Path, target.ClassifyCopy(targetEntry, registry)))
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

func active(options Options) (state.Source, domain.TargetScope, state.Registry, error) {
	selection, err := domain.SelectActive(domain.SelectionOptions{
		SourceID:    options.SourceID,
		Context:     options.Context,
		RequireHome: false,
	})
	if err != nil {
		return state.Source{}, domain.TargetScope{}, state.Registry{}, err
	}
	return selection.Source, selection.Scope, selection.Registry, nil
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

func entryFromCopyClass(targetPath string, selectedPackage string, selectedEntry string, class target.CopyClass) Entry {
	entry := Entry{
		TargetPath: targetPath,
		Package:    selectedPackage,
		Entry:      selectedEntry,
	}
	switch class.Kind {
	case target.CopyAbsent:
		entry.State = StateAbsent
	case target.CopyTrackedAbsent:
		entry.State = StateCopyMissing
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	case target.CopyUnchanged:
		entry.State = StateDeployed
	case target.CopySourceChanged:
		entry.State = string(target.CopySourceChanged)
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	case target.CopyTargetChanged:
		entry.State = string(target.CopyTargetChanged)
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	case target.CopyBothChanged:
		entry.State = string(target.CopyBothChanged)
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	case target.CopyOwnedOther:
		entry.State = StateOwnedByOther
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
		entry.Owner = class.Owner.Identity.String()
	default:
		entry.State = StateConflict
		entry.Code = class.ConflictCode()
		entry.Message = class.Message
	}
	return entry
}

func packageEntryFromCopyRecord(source state.Source, scope domain.TargetScope, record state.Copy) (target.PackageEntry, error) {
	pkg, err := packages.ResolveOne(source, scope.Context, record.Package)
	if err != nil {
		return target.PackageEntry{}, err
	}
	for _, entry := range packages.Leaves(pkg.Entries) {
		if entry.Rel == record.Path {
			return target.NewPackageEntry(pkg, entry, scope.LogicalRoot, scope.PhysicalPath)
		}
	}
	identity := packages.Identity{
		Source:  source.ID,
		Context: scope.Context,
		Name:    record.Package,
		Root:    filepath.Join(packages.Base(source, scope.Context), record.Package),
	}
	return target.PackageEntry{
		Identity:     identity,
		Entry:        packages.Entry{Path: filepath.Join(identity.Root, record.Path), Rel: record.Path, Deploy: packages.DeployCopy},
		PackageID:    identity.String(),
		ProviderKey:  identity.String() + ":" + record.Path,
		TargetPath:   record.Target,
		PhysicalPath: scope.PhysicalPath(record.Target),
	}, nil
}
