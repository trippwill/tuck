package pkgcmd

import (
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/target"
)

func buildUse(req UseRequest) (plan.Plan, error) {
	op, err := plan.NewOperation(plan.OperationOptions{
		Context:  req.Context,
		SourceID: req.SourceID,
		Apply:    req.Apply,
	})
	if err != nil {
		return plan.Plan{}, err
	}

	resolvedPackages, err := op.ResolvePackages(req.Refs, req.All)
	if err != nil {
		return plan.Plan{}, err
	}

	scope := op.Scope()
	plannedTargets := make(map[string]string)
	for _, pkg := range resolvedPackages {
		blockedRelPrefixes := make([]string, 0)
		for _, entry := range packages.Directories(pkg.Entries) {
			targetEntry, err := target.NewPackageEntry(pkg, entry, scope.LogicalRoot, scope.PhysicalPath)
			if err != nil {
				op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := targetEntry.Classify(op.Source(), scope.Context, scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				op.AddAction(plan.MkdirAction(targetEntry.TargetPath, targetEntry.PhysicalPath))
			case target.RealDirectory:
			default:
				op.AddConflict(plan.NewConflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
			}
		}

		for _, entry := range packages.Leaves(pkg.Entries) {
			if blockedByDirectoryConflict(entry.Rel, blockedRelPrefixes) {
				continue
			}
			targetEntry, err := target.NewPackageEntry(pkg, entry, scope.LogicalRoot, scope.PhysicalPath)
			if err != nil {
				op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetEntry.TargetPath]; ok && owner != targetEntry.ProviderKey {
				op.AddConflict(plan.NewConflict(target.ConflictMultipleProviders, targetEntry.TargetPath, targetEntry.PackageID, "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetEntry.TargetPath] = targetEntry.ProviderKey

			class := targetEntry.Classify(op.Source(), scope.Context, scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, &op)
				if ok {
					op.AddAction(action)
				}
			case target.ManagedSelected:
			default:
				op.AddConflict(plan.NewConflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}

	return op.Finalize()
}

func buildDrop(req DropRequest) (plan.Plan, error) {
	op, err := plan.NewOperation(plan.OperationOptions{
		Context:  req.Context,
		SourceID: req.SourceID,
		Apply:    req.Apply,
	})
	if err != nil {
		return plan.Plan{}, err
	}

	resolvedPackages, err := op.ResolvePackages(req.Refs, false)
	if err != nil {
		return plan.Plan{}, err
	}

	scope := op.Scope()
	for _, pkg := range resolvedPackages {
		for _, entry := range packages.Leaves(pkg.Entries) {
			targetEntry, err := target.NewPackageEntry(pkg, entry, scope.LogicalRoot, scope.PhysicalPath)
			if err != nil {
				op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				continue
			}

			class := targetEntry.Classify(op.Source(), scope.Context, scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
			case target.ManagedSelected:
				op.AddAction(plan.RemoveSymlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath))
			default:
				op.AddConflict(plan.NewConflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}

	return op.Finalize()
}

func buildRefresh(req RefreshRequest) (plan.Plan, error) {
	op, err := plan.NewOperation(plan.OperationOptions{
		Context:  req.Context,
		SourceID: req.SourceID,
		Apply:    req.Apply,
	})
	if err != nil {
		return plan.Plan{}, err
	}

	resolvedPackages, err := op.ResolvePackages(req.Refs, false)
	if err != nil {
		return plan.Plan{}, err
	}

	scope := op.Scope()
	removeActions := make([]plan.Action, 0)
	createActions := make([]plan.Action, 0)
	plannedTargets := make(map[string]string)
	for _, pkg := range resolvedPackages {
		blockedRelPrefixes := make([]string, 0)
		for _, entry := range packages.Directories(pkg.Entries) {
			targetEntry, err := target.NewPackageEntry(pkg, entry, scope.LogicalRoot, scope.PhysicalPath)
			if err != nil {
				op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := targetEntry.Classify(op.Source(), scope.Context, scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				createActions = append(createActions, plan.MkdirAction(targetEntry.TargetPath, targetEntry.PhysicalPath))
			case target.RealDirectory:
			default:
				op.AddConflict(plan.NewConflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
			}
		}

		for _, entry := range packages.Leaves(pkg.Entries) {
			if blockedByDirectoryConflict(entry.Rel, blockedRelPrefixes) {
				continue
			}
			targetEntry, err := target.NewPackageEntry(pkg, entry, scope.LogicalRoot, scope.PhysicalPath)
			if err != nil {
				op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetEntry.TargetPath]; ok && owner != targetEntry.ProviderKey {
				op.AddConflict(plan.NewConflict(target.ConflictMultipleProviders, targetEntry.TargetPath, targetEntry.PackageID, "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetEntry.TargetPath] = targetEntry.ProviderKey

			class := targetEntry.Classify(op.Source(), scope.Context, scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, &op)
				if ok {
					createActions = append(createActions, action)
				}
			case target.ManagedSelected:
				removeActions = append(removeActions, plan.RemoveSymlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath))
				action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, &op)
				if ok {
					createActions = append(createActions, action)
				}
			default:
				op.AddConflict(plan.NewConflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}

	if !op.HasConflicts() {
		op.AddActions(removeActions...)
		op.AddActions(createActions...)
	}
	return op.Finalize()
}

func symlinkAction(targetPath string, physicalTargetPath string, entryPath string, packageID string, op *plan.Operation) (plan.Action, bool) {
	payload, err := pathutil.SymlinkPayload(physicalTargetPath, entryPath)
	if err != nil {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetPath, packageID, err.Error()))
		return plan.Action{}, false
	}
	return plan.SymlinkAction(targetPath, physicalTargetPath, payload, entryPath), true
}

func blockedByDirectoryConflict(rel string, blockedPrefixes []string) bool {
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
