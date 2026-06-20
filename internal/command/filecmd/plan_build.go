package filecmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/state"
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

	packagePath, rel, err := pathutil.TargetToPackage(scope.LogicalRoot, targetPath, identity.Root)
	if err != nil {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, targetPath, identity.String(), err.Error()))
		return op.Finalize()
	}
	packagePathExists := false
	if _, err := os.Lstat(packagePath); err == nil {
		packagePathExists = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		return plan.Plan{}, apperr.AppErrWrapf(plan.ErrApply, err, "could not inspect package path %q", packagePath)
	}
	if !pathutil.Inside(packagePath, identity.Root) {
		op.AddConflict(plan.NewConflict(target.ConflictPathMismatch, packagePath, identity.String(), "package path escapes package root"))
		return op.Finalize()
	}
	deployCopy, mode := adoptDeployCopy(&op, identity, rel)
	if req.Copy {
		deployCopy = true
	}
	if deployCopy {
		targetMode, err := packages.ModeFromFile(physicalTargetPath)
		if err != nil {
			return plan.Plan{}, apperr.AppErrWrapf(plan.ErrApply, err, "could not inspect target mode %q", targetPath)
		}
		if req.SetMode {
			mode, err = packages.NormalizeModeFlag(req.Mode, targetMode)
			if err != nil {
				return plan.Plan{}, apperr.AppErrMsgf(plan.ErrApply, "invalid mode %q", req.Mode)
			}
		} else if mode == "" {
			mode = targetMode
		}
		configActions := []plan.Action{}
		if req.Copy {
			configAction, err := adoptCopyConfigAction(identity, rel, mode, req.SetMode)
			if err != nil {
				return plan.Plan{}, err
			}
			configActions = append(configActions, configAction)
		}
		if packagePathExists {
			record, ok := op.Registry().CopyByTarget(op.Source().ID, scope.Context, targetPath)
			if !ok || record.Package != identity.Name || record.Path != rel {
				op.AddConflict(plan.NewConflict(target.ConflictPackagePathExists, packagePath, identity.String(), "destination package path already exists"))
				return op.Finalize()
			}
			copyToPackage := plan.CopyAction(targetPath, physicalTargetPath, packagePath, "", mode, true, state.Copy{})
			copyBack, ok := trackedCopyFromTarget(identity, rel, packagePath, targetPath, physicalTargetPath, mode, true)
			if !ok {
				op.AddConflict(plan.NewConflict(target.ConflictGeneric, targetPath, identity.String(), "could not plan copy"))
				return op.Finalize()
			}
			op.AddActions(append([]plan.Action{copyToPackage, copyBack}, configActions...)...)
			return op.Finalize()
		}
		copyBack, ok := trackedCopyFromTarget(identity, rel, packagePath, targetPath, physicalTargetPath, mode, false)
		if !ok {
			op.AddConflict(plan.NewConflict(target.ConflictGeneric, targetPath, identity.String(), "could not plan copy"))
			return op.Finalize()
		}
		actions := []plan.Action{
			plan.MkdirAction(filepath.Dir(packagePath), ""),
			plan.MoveAction(targetPath, physicalTargetPath, packagePath, ""),
			copyBack,
		}
		op.AddActions(append(actions, configActions...)...)
		return op.Finalize()
	}
	if packagePathExists {
		op.AddConflict(plan.NewConflict(target.ConflictPackagePathExists, packagePath, identity.String(), "destination package path already exists"))
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
	if record, ok := op.Registry().CopyByTarget(op.Source().ID, scope.Context, targetPath); ok {
		copyEntry, err := copyEntryFromRecord(op.Source(), scope.Context, record, scope.LogicalRoot, scope.PhysicalPath)
		if err != nil {
			op.AddConflict(plan.NewConflict(target.ConflictGeneric, targetPath, "", err.Error()))
			return op.Finalize()
		}
		copyClass := target.ClassifyCopy(copyEntry, op.Registry())
		if copyClass.Kind != target.CopyUnchanged {
			op.SetPackages(copyEntry.PackageID)
			op.AddConflict(plan.NewConflict(copyClass.ConflictCode(), targetPath, copyEntry.PackageID, copyMessage(copyClass)))
			return op.Finalize()
		}
		op.SetPackages(copyEntry.PackageID)
		op.AddActions(
			plan.ForgetCopyAction(targetPath, record),
			plan.RemoveFileAction(copyEntry.Entry.Path, ""),
		)
		pruneDirs, err := pruneAfterEject(copyEntry.Identity.Root, copyEntry.Entry.Path)
		if err != nil {
			return plan.Plan{}, err
		}
		for _, dir := range pruneDirs {
			op.AddAction(plan.RmdirAction(dir, ""))
		}
		return op.Finalize()
	}
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

func adoptDeployCopy(op *plan.Operation, identity packages.Identity, rel string) (bool, string) {
	if record, ok := op.Registry().CopyByEntry(identity.Source, identity.Context, identity.Name, rel); ok && record.Target != "" {
		return true, record.TargetMode
	}

	entries, err := packages.Enumerate(identity.Root)
	if err != nil {
		return false, ""
	}
	config, err := packages.LoadConfig(identity.Root, entries)
	if err != nil {
		return false, ""
	}
	if file, ok := packages.ConfiguredFile(config, rel); ok && file.Deploy == packages.DeployCopy {
		return true, file.Mode
	}
	return false, ""
}

func adoptCopyConfigAction(identity packages.Identity, rel string, mode string, setMode bool) (plan.Action, error) {
	var config packages.PackageConfig
	entries, err := packages.Enumerate(identity.Root)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return plan.Action{}, err
		}
	} else {
		config, err = packages.LoadConfig(identity.Root, entries)
		if err != nil {
			return plan.Action{}, err
		}
	}
	file, ok := packages.ConfiguredFile(config, rel)
	if !ok {
		file = packages.FileConfig{Path: rel}
	}
	file.Deploy = packages.DeployCopy
	if setMode {
		file.Mode = mode
	}
	config = packages.SetFileConfig(config, file)
	manifestPath := filepath.Join(identity.Root, manifest.ManifestFilename)
	return plan.PackageConfigAction(manifestPath, identity.Root, config), nil
}

func trackedCopyFromTarget(identity packages.Identity, rel string, packagePath string, targetPath string, physicalTargetPath string, mode string, overwrite bool) (plan.Action, bool) {
	checksum, err := state.FileChecksum(physicalTargetPath)
	if err != nil {
		return plan.Action{}, false
	}
	record := state.Copy{
		Source:         identity.Source,
		Context:        identity.Context,
		Package:        identity.Name,
		Path:           rel,
		Target:         targetPath,
		SourceChecksum: checksum,
		TargetChecksum: checksum,
		TargetMode:     mode,
	}
	return plan.CopyAction(packagePath, "", targetPath, physicalTargetPath, mode, overwrite, record), true
}

func copyEntryFromRecord(source state.Source, contextName string, record state.Copy, logicalRoot string, physicalPath func(string) string) (target.PackageEntry, error) {
	if pkg, err := packages.ResolveOne(source, contextName, record.Package); err == nil {
		for _, entry := range packages.Leaves(pkg.Entries) {
			if entry.Rel == record.Path {
				return target.NewPackageEntry(pkg, entry, logicalRoot, physicalPath)
			}
		}
	}
	identity := packages.Identity{
		Source:  source.ID,
		Context: contextName,
		Name:    record.Package,
		Root:    filepath.Join(packages.Base(source, contextName), record.Package),
	}
	entry := packages.Entry{Path: filepath.Join(identity.Root, record.Path), Rel: record.Path, Deploy: packages.DeployCopy}
	return target.PackageEntry{
		Identity:     identity,
		Entry:        entry,
		PackageID:    identity.String(),
		ProviderKey:  identity.String() + ":" + entry.Rel,
		TargetPath:   record.Target,
		PhysicalPath: physicalPath(record.Target),
	}, nil
}

func copyMessage(class target.CopyClass) string {
	if class.Message != "" {
		return class.Message
	}
	return "copied target is not safe to modify"
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
