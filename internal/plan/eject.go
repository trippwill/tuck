package plan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/target"
)

type EjectOptions struct {
	File       string
	SourceID   string
	Context    string
	TargetRoot string
	Apply      bool
}

func BuildEject(options EjectOptions) (UsePlan, error) {
	op, err := newOperation(operationOptions{
		Command:    "eject",
		Context:    options.Context,
		TargetRoot: options.TargetRoot,
		SourceID:   options.SourceID,
		Apply:      options.Apply,
	})
	if err != nil {
		return UsePlan{}, err
	}
	targetPath, err := pathutil.ExpandInput(options.File)
	if err != nil {
		return UsePlan{}, AppErrWrapf(ErrApply, err, "could not resolve target path")
	}

	physicalTargetPath := op.scope.PhysicalPath(targetPath)
	class := target.ClassifyAt(targetPath, physicalTargetPath, op.source, op.scope.Context, op.scope.LogicalRoot, nil, "")
	if class.Kind == target.PathMismatch {
		op.plan.Packages = []string{class.Owner.Identity.String()}
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, class.Owner.Identity.String(), class.Message))
		return op.finalize()
	}
	if class.Kind != target.Managed {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictNotManagedSymlink, targetPath, "", notManagedMessage(class)))
		return op.finalize()
	}

	owner := class.Owner
	op.plan.Packages = []string{owner.Identity.String()}
	if filepath.Clean(targetPath) != filepath.Clean(owner.ExpectedTarget) {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, owner.Identity.String(), "managed symlink path does not match package entry"))
		return op.finalize()
	}

	packagePath := owner.EntryPath
	info, err := os.Lstat(packagePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictNotManagedSymlink, packagePath, owner.Identity.String(), "package file does not exist"))
			return op.finalize()
		}
		return UsePlan{}, AppErrWrapf(ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if info.IsDir() {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictNotManagedSymlink, packagePath, owner.Identity.String(), "package path is a directory"))
		return op.finalize()
	}

	op.plan.Actions = append(op.plan.Actions,
		Action{Type: ActionRemoveSymlink, Path: targetPath, physicalPath: physicalTargetPath},
		Action{Type: ActionMove, Src: packagePath, Dst: targetPath, physicalDst: physicalTargetPath},
	)
	pruneDirs, err := pruneAfterEject(owner.Identity.Root, packagePath)
	if err != nil {
		return UsePlan{}, err
	}
	for _, dir := range pruneDirs {
		op.plan.Actions = append(op.plan.Actions, Action{Type: ActionRmdir, Path: dir})
	}
	return op.finalize()
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
			return nil, AppErrMsgf(ErrApply, "package path %q escapes package root %q", packagePath, packageRoot)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, AppErrWrapf(ErrApply, err, "could not inspect package directory %q", dir)
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
