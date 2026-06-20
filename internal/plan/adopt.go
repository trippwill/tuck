package plan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/target"
)

type AdoptOptions struct {
	File       string
	Ref        string
	SourceID   string
	Context    string
	TargetRoot string
	Apply      bool
}

func BuildAdopt(options AdoptOptions) (Plan, error) {
	op, err := newOperation(operationOptions{
		Context:    options.Context,
		TargetRoot: options.TargetRoot,
		SourceID:   options.SourceID,
		Apply:      options.Apply,
	})
	if err != nil {
		return Plan{}, err
	}
	identity, err := packages.ResolveIdentity(op.source, op.scope.Context, options.Ref)
	if err != nil {
		return Plan{}, err
	}
	op.plan.Packages = []string{identity.String()}

	targetPath, err := pathutil.ExpandInput(options.File)
	if err != nil {
		return Plan{}, AppErrWrapf(ErrApply, err, "could not resolve target path")
	}
	if !pathutil.Inside(targetPath, op.scope.LogicalRoot) {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictOutsideTargetRoot, targetPath, identity.String(), "target is outside the selected target root"))
		return op.finalize()
	}
	for _, enabled := range op.registry.EnabledSources() {
		if pathutil.Inside(targetPath, enabled.Path) {
			op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictInsideSourceRepo, targetPath, identity.String(), "target is inside an enabled source repository"))
			return op.finalize()
		}
	}

	physicalTargetPath := op.scope.PhysicalPath(targetPath)
	class := target.ClassifyAt(targetPath, physicalTargetPath, op.source, op.scope.Context, op.scope.LogicalRoot, nil, "")
	if class.Kind != target.RealFile {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(adoptConflictCode(class), targetPath, identity.String(), adoptConflictMessage(class)))
		return op.finalize()
	}

	packagePath, _, err := pathutil.TargetToPackage(op.scope.LogicalRoot, targetPath, identity.Root)
	if err != nil {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return op.finalize()
	}
	if _, err := os.Lstat(packagePath); err == nil {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPackagePathExists, packagePath, identity.String(), "destination package path already exists"))
		return op.finalize()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Plan{}, AppErrWrapf(ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if !pathutil.Inside(packagePath, identity.Root) {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, packagePath, identity.String(), "package path escapes package root"))
		return op.finalize()
	}
	payload, err := pathutil.SymlinkPayload(physicalTargetPath, packagePath)
	if err != nil {
		op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return op.finalize()
	}

	op.plan.Actions = append(op.plan.Actions,
		Action{Type: ActionMkdir, Path: filepath.Dir(packagePath)},
		Action{Type: ActionMove, Src: targetPath, physicalSrc: physicalTargetPath, Dst: packagePath},
		Action{Type: ActionSymlink, LinkPath: targetPath, physicalLinkPath: physicalTargetPath, Payload: payload, Target: packagePath},
	)

	return op.finalize()
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
