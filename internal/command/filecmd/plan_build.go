package filecmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/target"
)

func buildAdopt(req AdoptRequest) (plan.Plan, error) {
	op, err := plan.NewOperation(plan.OperationOptions{
		Context:  req.Context,
		SourceID: req.SourceID,
		Apply:    req.Apply,
	})
	if err != nil {
		return plan.Plan{}, err
	}
	identity, err := packages.ResolveIdentity(op.Source(), op.Scope().Context, req.Ref)
	if err != nil {
		return plan.Plan{}, err
	}
	op.SetPackages(identity.String())

	targetPath, err := pathutil.ExpandInput(req.File)
	if err != nil {
		return plan.Plan{}, apperr.AppErrWrapf(plan.ErrApply, err, "could not resolve target path")
	}
	scope := op.Scope()
	if !pathutil.Inside(targetPath, scope.LogicalRoot) {
		op.AddConflict(plan.NewConflict(target.ConflictOutsideTargetRoot, targetPath, identity.String(), "target is outside the selected target root"))
		return op.Finalize()
	}
	for _, enabled := range op.Registry().EnabledSources() {
		if pathutil.Inside(targetPath, enabled.Path) {
			op.AddConflict(plan.NewConflict(target.ConflictInsideSourceRepo, targetPath, identity.String(), "target is inside an enabled source repository"))
			return op.Finalize()
		}
	}

	physicalTargetPath := scope.PhysicalPath(targetPath)
	class := target.ClassifyAt(targetPath, physicalTargetPath, op.Source(), scope.Context, scope.LogicalRoot, nil, "")
	if class.Kind != target.RealFile {
		op.AddConflict(plan.NewConflict(adoptConflictCode(class), targetPath, identity.String(), adoptConflictMessage(class)))
		return op.Finalize()
	}

	packagePath, _, err := pathutil.TargetToPackage(scope.LogicalRoot, targetPath, identity.Root)
	if err != nil {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return op.Finalize()
	}
	if _, err := os.Lstat(packagePath); err == nil {
		op.AddConflict(plan.NewConflict(target.ConflictPackagePathExists, packagePath, identity.String(), "destination package path already exists"))
		return op.Finalize()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return plan.Plan{}, apperr.AppErrWrapf(plan.ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if !pathutil.Inside(packagePath, identity.Root) {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, packagePath, identity.String(), "package path escapes package root"))
		return op.Finalize()
	}
	payload, err := pathutil.SymlinkPayload(physicalTargetPath, packagePath)
	if err != nil {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return op.Finalize()
	}

	op.AddActions(
		plan.MkdirAction(filepath.Dir(packagePath), ""),
		plan.MoveAction(targetPath, physicalTargetPath, packagePath, ""),
		plan.SymlinkAction(targetPath, physicalTargetPath, payload, packagePath),
	)

	return op.Finalize()
}

func buildEject(req EjectRequest) (plan.Plan, error) {
	op, err := plan.NewOperation(plan.OperationOptions{
		Context:  req.Context,
		SourceID: req.SourceID,
		Apply:    req.Apply,
	})
	if err != nil {
		return plan.Plan{}, err
	}
	targetPath, err := pathutil.ExpandInput(req.File)
	if err != nil {
		return plan.Plan{}, apperr.AppErrWrapf(plan.ErrApply, err, "could not resolve target path")
	}

	scope := op.Scope()
	physicalTargetPath := scope.PhysicalPath(targetPath)
	class := target.ClassifyAt(targetPath, physicalTargetPath, op.Source(), scope.Context, scope.LogicalRoot, nil, "")
	if class.Kind == target.PathMismatch {
		op.SetPackages(class.Owner.Identity.String())
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetPath, class.Owner.Identity.String(), class.Message))
		return op.Finalize()
	}
	if class.Kind != target.Managed {
		op.AddConflict(plan.NewConflict(target.ConflictNotManagedSymlink, targetPath, "", notManagedMessage(class)))
		return op.Finalize()
	}

	owner := class.Owner
	op.SetPackages(owner.Identity.String())
	if filepath.Clean(targetPath) != filepath.Clean(owner.ExpectedTarget) {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetPath, owner.Identity.String(), "managed symlink path does not match package entry"))
		return op.Finalize()
	}

	packagePath := owner.EntryPath
	info, err := os.Lstat(packagePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			op.AddConflict(plan.NewConflict(target.ConflictNotManagedSymlink, packagePath, owner.Identity.String(), "package file does not exist"))
			return op.Finalize()
		}
		return plan.Plan{}, apperr.AppErrWrapf(plan.ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if info.IsDir() {
		op.AddConflict(plan.NewConflict(target.ConflictNotManagedSymlink, packagePath, owner.Identity.String(), "package path is a directory"))
		return op.Finalize()
	}

	op.AddActions(
		plan.RemoveSymlinkAction(targetPath, physicalTargetPath),
		plan.MoveAction(packagePath, "", targetPath, physicalTargetPath),
	)
	pruneDirs, err := pruneAfterEject(owner.Identity.Root, packagePath)
	if err != nil {
		return plan.Plan{}, err
	}
	for _, dir := range pruneDirs {
		op.AddAction(plan.RmdirAction(dir, ""))
	}
	return op.Finalize()
}

func adoptConflictCode(class target.Class) target.ConflictCode {
	if class.Kind == target.Absent {
		return target.ConflictAbsent
	}
	return class.ConflictCode()
}

func adoptConflictMessage(class target.Class) string {
	if class.Message != "" {
		return class.Message
	}
	if class.Kind == target.Absent {
		return "target does not exist"
	}
	return "target is not a real file"
}

func notManagedMessage(class target.Class) string {
	if class.Message != "" {
		return class.Message
	}
	return "target is not a managed symlink"
}

func pruneAfterEject(packageRoot, packagePath string) ([]string, error) {
	packageRoot = filepath.Clean(packageRoot)
	packagePath = filepath.Clean(packagePath)
	pruneDirs := make([]string, 0)
	plannedRemoved := map[string]struct{}{
		packagePath: {},
	}

	for dir := filepath.Dir(packagePath); dir != packageRoot; dir = filepath.Dir(dir) {
		if !pathutil.Inside(dir, packageRoot) {
			return nil, apperr.AppErrMsgf(plan.ErrApply, "package path %q escapes package root %q", packagePath, packageRoot)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, apperr.AppErrWrapf(plan.ErrApply, err, "could not inspect package directory %q", dir)
		}
		emptyAfterPlannedRemovals := true
		for _, entry := range entries {
			entryPath := filepath.Join(dir, entry.Name())
			if _, ok := plannedRemoved[entryPath]; ok {
				continue
			}
			emptyAfterPlannedRemovals = false
			break
		}
		if !emptyAfterPlannedRemovals {
			break
		}
		pruneDirs = append(pruneDirs, dir)
		plannedRemoved[dir] = struct{}{}
	}
	return pruneDirs, nil
}
