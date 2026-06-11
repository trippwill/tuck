package plan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
	"github.com/trippwill/tuck/internal/target"
)

type AdoptOptions struct {
	File       string
	Ref        string
	SourceID   string
	TargetRoot string
	Apply      bool
}

func BuildAdopt(options AdoptOptions) (UsePlan, error) {
	targetRoot, err := domain.TargetRoot(options.TargetRoot, true)
	if err != nil {
		return UsePlan{}, AppErrMsg(ErrApply, err.Error())
	}
	registry, err := state.Load()
	if err != nil {
		return UsePlan{}, err
	}
	source, err := resolve.ActiveSource(registry, options.SourceID)
	if err != nil {
		return UsePlan{}, err
	}
	identity, err := packages.ResolveForAdopt(source, packages.ContextHome, options.Ref)
	if err != nil {
		return UsePlan{}, err
	}

	adoptPlan := UsePlan{
		Command:   "adopt",
		Context:   packages.ContextHome,
		DryRun:    !options.Apply,
		Applied:   false,
		Packages:  []string{identity.String()},
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}

	targetPath, err := pathutil.ExpandInput(options.File)
	if err != nil {
		return UsePlan{}, AppErrWrapf(ErrApply, err, "could not resolve target path")
	}
	if !pathutil.Inside(targetPath, targetRoot) {
		adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(target.ConflictOutsideTargetRoot, targetPath, identity.String(), "target is outside the selected target root"))
		return adoptPlan, nil
	}
	for _, enabled := range registry.EnabledSources() {
		if pathutil.Inside(targetPath, enabled.Path) {
			adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(target.ConflictInsideSourceRepo, targetPath, identity.String(), "target is inside an enabled source repository"))
			return adoptPlan, nil
		}
	}

	class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, nil, "")
	if class.Kind != target.RealFile {
		adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(adoptConflictCode(class), targetPath, identity.String(), adoptConflictMessage(class)))
		return adoptPlan, nil
	}

	packagePath, _, err := pathutil.TargetToPackage(targetRoot, targetPath, identity.Root)
	if err != nil {
		adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return adoptPlan, nil
	}
	if _, err := os.Lstat(packagePath); err == nil {
		adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(target.ConflictPackagePathExists, packagePath, identity.String(), "destination package path already exists"))
		return adoptPlan, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return UsePlan{}, AppErrWrapf(ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if !pathutil.Inside(packagePath, identity.Root) {
		adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(target.ConflictPathMismatch, packagePath, identity.String(), "package path escapes package root"))
		return adoptPlan, nil
	}
	payload, err := pathutil.SymlinkPayload(targetPath, packagePath)
	if err != nil {
		adoptPlan.Conflicts = append(adoptPlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return adoptPlan, nil
	}

	adoptPlan.Actions = append(adoptPlan.Actions,
		Action{Type: ActionMkdir, Path: filepath.Dir(packagePath)},
		Action{Type: ActionMove, Src: targetPath, Dst: packagePath},
		Action{Type: ActionSymlink, LinkPath: targetPath, Payload: payload, Target: packagePath},
	)

	if options.Apply {
		if err := Apply(adoptPlan); err != nil {
			return adoptPlan, err
		}
		adoptPlan.Applied = true
		adoptPlan.DryRun = false
	}
	return adoptPlan, nil
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
