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

type EjectOptions struct {
	File       string
	SourceID   string
	TargetRoot string
	Apply      bool
}

func BuildEject(options EjectOptions) (UsePlan, error) {
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
	targetPath, err := pathutil.ExpandInput(options.File)
	if err != nil {
		return UsePlan{}, AppErrWrapf(ErrApply, err, "could not resolve target path")
	}

	ejectPlan := UsePlan{
		Command:   "eject",
		Context:   packages.ContextHome,
		DryRun:    !options.Apply,
		Applied:   false,
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}

	class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, nil, "")
	if class.Kind == target.PathMismatch {
		ejectPlan.Packages = []string{class.Owner.Identity.String()}
		ejectPlan.Conflicts = append(ejectPlan.Conflicts, conflict("path_mismatch", targetPath, class.Owner.Identity.String(), class.Message))
		return ejectPlan, nil
	}
	if class.Kind != target.Managed {
		ejectPlan.Conflicts = append(ejectPlan.Conflicts, conflict("not_a_managed_symlink", targetPath, "", notManagedMessage(class)))
		return ejectPlan, nil
	}

	owner := class.Owner
	ejectPlan.Packages = []string{owner.Identity.String()}
	if filepath.Clean(targetPath) != filepath.Clean(owner.ExpectedTarget) {
		ejectPlan.Conflicts = append(ejectPlan.Conflicts, conflict("path_mismatch", targetPath, owner.Identity.String(), "managed symlink path does not match package entry"))
		return ejectPlan, nil
	}

	packagePath := owner.EntryPath
	info, err := os.Lstat(packagePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			ejectPlan.Conflicts = append(ejectPlan.Conflicts, conflict("not_a_managed_symlink", packagePath, owner.Identity.String(), "package file does not exist"))
			return ejectPlan, nil
		}
		return UsePlan{}, AppErrWrapf(ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if info.IsDir() {
		ejectPlan.Conflicts = append(ejectPlan.Conflicts, conflict("not_a_managed_symlink", packagePath, owner.Identity.String(), "package path is a directory"))
		return ejectPlan, nil
	}

	ejectPlan.Actions = append(ejectPlan.Actions,
		Action{Type: "remove_symlink", Path: targetPath},
		Action{Type: "move", Src: packagePath, Dst: targetPath},
	)
	pruneDirs, err := pruneAfterEject(owner.Identity.Root, packagePath)
	if err != nil {
		return UsePlan{}, err
	}
	for _, dir := range pruneDirs {
		ejectPlan.Actions = append(ejectPlan.Actions, Action{Type: "rmdir", Path: dir})
	}
	if options.Apply {
		if err := Apply(ejectPlan); err != nil {
			return ejectPlan, err
		}
		ejectPlan.Applied = true
		ejectPlan.DryRun = false
	}
	return ejectPlan, nil
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
