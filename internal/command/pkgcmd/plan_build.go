package pkgcmd

import (
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/state"
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

	_, createActions := packageCreateActions(&op, resolvedPackages, false)
	op.AddActions(createActions...)

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
			if _, tracked := op.Registry().CopyByEntry(pkg.Identity.Source, pkg.Identity.Context, pkg.Identity.Name, entry.Rel); tracked {
				copyClass := target.ClassifyCopy(targetEntry, op.Registry())
				switch copyClass.Kind {
				case target.CopyTrackedAbsent, target.CopyUnchanged, target.CopySourceChanged:
					op.AddAction(plan.RemoveCopyAction(targetEntry.TargetPath, targetEntry.PhysicalPath, copyClass.Record))
				case target.CopyTargetChanged, target.CopyBothChanged:
					op.AddConflict(plan.NewConflict(copyClass.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, copyClass.Message))
				default:
					op.AddConflict(plan.NewConflict(copyClass.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, copyMessage(copyClass)))
				}
				continue
			}
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

	removeActions, createActions := packageCreateActions(&op, resolvedPackages, true)
	if !op.HasConflicts() {
		op.AddActions(removeActions...)
		op.AddActions(createActions...)
	}
	return op.Finalize()
}

func packageCreateActions(op *plan.Operation, resolvedPackages []packages.Resolved, refresh bool) ([]plan.Action, []plan.Action) {
	scope := op.Scope()
	removeActions := []plan.Action{}
	createActions := []plan.Action{}
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

			if copyClass, tracked := refreshCopyRemoval(op, targetEntry, refresh); tracked {
				if copyClass.Kind == target.CopyTargetChanged || copyClass.Kind == target.CopyBothChanged {
					op.AddConflict(plan.NewConflict(copyClass.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, copyClass.Message))
					continue
				}
				removeActions = append(removeActions, plan.RemoveCopyAction(targetEntry.TargetPath, targetEntry.PhysicalPath, copyClass.Record))
				action, ok := deployAction(targetEntry, op, true)
				if ok {
					createActions = append(createActions, action)
				}
				continue
			}

			if targetEntry.Entry.Deploy == packages.DeployCopy {
				copyClass := target.ClassifyCopy(targetEntry, op.Registry())
				switch copyClass.Kind {
				case target.CopyAbsent, target.CopyTrackedAbsent:
					action, ok := copyAction(targetEntry, op, false)
					if ok {
						createActions = append(createActions, action)
					}
				case target.CopyUnchanged:
				default:
					op.AddConflict(plan.NewConflict(copyClass.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, copyMessage(copyClass)))
				}
				continue
			}

			class := targetEntry.Classify(op.Source(), scope.Context, scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, op)
				if ok {
					createActions = append(createActions, action)
				}
			case target.ManagedSelected:
				if refresh {
					removeActions = append(removeActions, plan.RemoveSymlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath))
					action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, op)
					if ok {
						createActions = append(createActions, action)
					}
				}
			default:
				op.AddConflict(plan.NewConflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}
	return removeActions, createActions
}

func refreshCopyRemoval(op *plan.Operation, targetEntry target.PackageEntry, refresh bool) (target.CopyClass, bool) {
	if !refresh {
		return target.CopyClass{}, false
	}
	if _, tracked := op.Registry().CopyByEntry(targetEntry.Identity.Source, targetEntry.Identity.Context, targetEntry.Identity.Name, targetEntry.Entry.Rel); !tracked {
		return target.CopyClass{}, false
	}
	return target.ClassifyCopy(targetEntry, op.Registry()), true
}

func deployAction(targetEntry target.PackageEntry, op *plan.Operation, overwrite bool) (plan.Action, bool) {
	if targetEntry.Entry.Deploy == packages.DeployCopy {
		return copyAction(targetEntry, op, overwrite)
	}
	return symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, op)
}

func copyAction(targetEntry target.PackageEntry, op *plan.Operation, overwrite bool) (plan.Action, bool) {
	sourceChecksum, err := state.FileChecksum(targetEntry.Entry.Path)
	if err != nil {
		op.AddConflict(plan.NewConflict(target.ConflictGeneric, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
		return plan.Action{}, false
	}
	mode := targetEntry.Entry.Mode
	if mode == "" {
		mode, err = packages.ModeFromFile(targetEntry.Entry.Path)
		if err != nil {
			op.AddConflict(plan.NewConflict(target.ConflictGeneric, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
			return plan.Action{}, false
		}
	}
	copyRecord := state.Copy{
		Source:         targetEntry.Identity.Source,
		Context:        targetEntry.Identity.Context,
		Package:        targetEntry.Identity.Name,
		Path:           targetEntry.Entry.Rel,
		Target:         targetEntry.TargetPath,
		SourceChecksum: sourceChecksum,
		TargetChecksum: sourceChecksum,
		TargetMode:     mode,
	}
	return plan.CopyAction(targetEntry.Entry.Path, "", targetEntry.TargetPath, targetEntry.PhysicalPath, mode, overwrite, copyRecord), true
}

func copyMessage(class target.CopyClass) string {
	if class.Message != "" {
		return class.Message
	}
	return "copied target is not safe to modify"
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
