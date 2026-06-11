package plan

import (
	"os"
	"path/filepath"
)

func Apply(usePlan UsePlan) error {
	if len(usePlan.Conflicts) > 0 {
		return AppErrMsg(ErrApply, "cannot apply a plan with conflicts")
	}
	// Preflight catches predictable failures before the first mutation. The
	// execution switch still checks every action because the filesystem can
	// change between validation and mutation.
	if err := preflightApply(usePlan); err != nil {
		return err
	}
	for _, action := range usePlan.Actions {
		switch action.Type {
		case ActionMkdir:
			if err := os.MkdirAll(action.Path, 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", action.Path)
			}
		case ActionRmdir:
			info, err := os.Lstat(action.Path)
			if err != nil {
				return AppErrWrapf(ErrApply, err, "could not inspect directory %q", action.Path)
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return AppErrMsgf(ErrApply, "refusing to remove non-directory %q", action.Path)
			}
			if err := os.Remove(action.Path); err != nil {
				return AppErrWrapf(ErrApply, err, "could not remove directory %q", action.Path)
			}
		case ActionSymlink:
			if err := os.MkdirAll(filepath.Dir(action.LinkPath), 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", filepath.Dir(action.LinkPath))
			}
			if err := os.Symlink(action.Payload, action.LinkPath); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create symlink %q", action.LinkPath)
			}
		case ActionRemoveSymlink:
			info, err := os.Lstat(action.Path)
			if err != nil {
				return AppErrWrapf(ErrApply, err, "could not inspect symlink %q", action.Path)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return AppErrMsgf(ErrApply, "refusing to remove non-symlink %q", action.Path)
			}
			if err := os.Remove(action.Path); err != nil {
				return AppErrWrapf(ErrApply, err, "could not remove symlink %q", action.Path)
			}
		case ActionMove:
			if err := os.Rename(action.Src, action.Dst); err != nil {
				return AppErrWrapf(ErrApply, err, "could not move %q to %q", action.Src, action.Dst)
			}
		default:
			return AppErrMsgf(ErrApply, "unknown plan action type %q", action.Type)
		}
	}
	return nil
}

type preflightPathKind int

const (
	// preflightRemoved means the path is absent after earlier planned actions,
	// even if it exists on disk before Apply starts.
	preflightRemoved preflightPathKind = iota
	// preflightDirectory means the path is or will be a real directory.
	preflightDirectory
	// preflightNonDirectory means the path is or will be a file-like entry.
	preflightNonDirectory
	// preflightSymlink means the path is or will be a symlink.
	preflightSymlink
)

type applyPreflight struct {
	paths map[string]preflightPathKind
}

// preflightApply validates the whole ordered action list before any mutation.
// It records each validated action into a small planned-state overlay so later
// validations see earlier actions as already applied. This keeps valid sequences
// such as mkdir-then-symlink, remove_symlink-then-move, and move-then-rmdir from
// being rejected just because the initial filesystem state does not yet match
// the later action's requirements.
func preflightApply(usePlan UsePlan) error {
	preflight := applyPreflight{paths: make(map[string]preflightPathKind)}
	for _, action := range usePlan.Actions {
		if err := preflight.validate(action); err != nil {
			return err
		}
		preflight.record(action)
	}
	return nil
}

func (p *applyPreflight) validate(action Action) error {
	switch action.Type {
	case ActionMkdir:
		return p.validateMkdir(action.Path)
	case ActionRmdir:
		return p.validateRmdir(action.Path)
	case ActionSymlink:
		return p.validateSymlink(action)
	case ActionRemoveSymlink:
		return p.validateRemoveSymlink(action.Path)
	case ActionMove:
		return p.validateMove(action)
	default:
		return AppErrMsgf(ErrApply, "unknown plan action type %q", action.Type)
	}
}

func (p *applyPreflight) record(action Action) {
	switch action.Type {
	case ActionMkdir:
		p.recordMkdir(action.Path)
	case ActionRmdir:
		p.paths[filepath.Clean(action.Path)] = preflightRemoved
	case ActionSymlink:
		p.paths[filepath.Clean(action.LinkPath)] = preflightSymlink
	case ActionRemoveSymlink:
		p.paths[filepath.Clean(action.Path)] = preflightRemoved
	case ActionMove:
		p.paths[filepath.Clean(action.Src)] = preflightRemoved
		p.paths[filepath.Clean(action.Dst)] = preflightNonDirectory
	}
}

func (p *applyPreflight) validateMkdir(path string) error {
	path = filepath.Clean(path)
	for dir := path; ; dir = filepath.Dir(dir) {
		exists, isDir, err := p.directoryState(dir)
		if err != nil {
			return AppErrWrapf(ErrApply, err, "could not inspect directory %q", dir)
		}
		if exists {
			if !isDir {
				return AppErrMsgf(ErrApply, "could not create directory %q: %q is not a directory", path, dir)
			}
			return nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			return nil
		}
	}
}

func (p *applyPreflight) validateRmdir(path string) error {
	path = filepath.Clean(path)
	kind, exists, err := p.pathKind(path)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect directory %q", path)
	}
	if !exists {
		return AppErrMsgf(ErrApply, "could not inspect directory %q: path does not exist", path)
	}
	if kind != preflightDirectory {
		return AppErrMsgf(ErrApply, "refusing to remove non-directory %q", path)
	}
	empty, err := p.emptyDirectoryAfterPlannedRemovals(path)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect directory %q", path)
	}
	if !empty {
		return AppErrMsgf(ErrApply, "could not remove directory %q: directory not empty", path)
	}
	return nil
}

func (p *applyPreflight) validateSymlink(action Action) error {
	parent := filepath.Dir(action.LinkPath)
	if err := p.validateMkdir(parent); err != nil {
		return err
	}
	_, exists, err := p.pathKind(action.LinkPath)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect symlink %q", action.LinkPath)
	}
	if exists {
		return AppErrMsgf(ErrApply, "could not create symlink %q: path already exists", action.LinkPath)
	}
	return nil
}

func (p *applyPreflight) validateRemoveSymlink(path string) error {
	path = filepath.Clean(path)
	kind, exists, err := p.pathKind(path)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect symlink %q", path)
	}
	if !exists {
		return AppErrMsgf(ErrApply, "could not inspect symlink %q: path does not exist", path)
	}
	if kind != preflightSymlink {
		return AppErrMsgf(ErrApply, "refusing to remove non-symlink %q", path)
	}
	return nil
}

func (p *applyPreflight) validateMove(action Action) error {
	_, exists, err := p.pathKind(action.Src)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect move source %q", action.Src)
	}
	if !exists {
		return AppErrMsgf(ErrApply, "could not move %q to %q: source does not exist", action.Src, action.Dst)
	}
	_, exists, err = p.pathKind(action.Dst)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect move destination %q", action.Dst)
	}
	if exists {
		return AppErrMsgf(ErrApply, "could not move %q to %q: destination already exists", action.Src, action.Dst)
	}
	if err := p.validateExistingDirectory(filepath.Dir(action.Dst)); err != nil {
		return err
	}
	return nil
}

func (p *applyPreflight) validateExistingDirectory(path string) error {
	path = filepath.Clean(path)
	exists, isDir, err := p.directoryState(path)
	if err != nil {
		return AppErrWrapf(ErrApply, err, "could not inspect directory %q", path)
	}
	if !exists {
		return AppErrMsgf(ErrApply, "directory %q does not exist", path)
	}
	if !isDir {
		return AppErrMsgf(ErrApply, "%q is not a directory", path)
	}
	return nil
}

// directoryState follows symlinks, matching operations that need a usable parent
// directory such as os.MkdirAll and os.Rename. Symlink-removal and rmdir checks
// use pathKind instead because they must inspect the path itself.
func (p *applyPreflight) directoryState(path string) (bool, bool, error) {
	path = filepath.Clean(path)
	if kind, ok := p.paths[path]; ok {
		return kind != preflightRemoved, kind == preflightDirectory, nil
	}
	if p.hasRemovedAncestor(path) {
		return false, false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, false, err
		}
		if linkInfo, linkErr := os.Lstat(path); linkErr == nil && linkInfo.Mode()&os.ModeSymlink != 0 {
			return true, false, nil
		}
		return false, false, nil
	}
	return true, info.IsDir(), nil
}

// pathKind uses Lstat semantics because most plan actions operate on the exact
// directory entry they name: remove_symlink must see a symlink, rmdir must see a
// real directory, and symlink creation must fail if any entry already exists.
func (p *applyPreflight) pathKind(path string) (preflightPathKind, bool, error) {
	path = filepath.Clean(path)
	if kind, ok := p.paths[path]; ok {
		return kind, kind != preflightRemoved, nil
	}
	if p.hasRemovedAncestor(path) {
		return preflightRemoved, false, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return preflightRemoved, false, nil
		}
		return preflightRemoved, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return preflightSymlink, true, nil
	}
	if info.IsDir() {
		return preflightDirectory, true, nil
	}
	return preflightNonDirectory, true, nil
}

// recordMkdir marks only the missing directories that mkdir would create; it
// stops at the first existing directory in the planned-state view.
func (p *applyPreflight) recordMkdir(path string) {
	path = filepath.Clean(path)
	for dir := path; ; dir = filepath.Dir(dir) {
		exists, _, _ := p.directoryState(dir)
		if exists {
			return
		}
		p.paths[dir] = preflightDirectory
		if parent := filepath.Dir(dir); parent == dir {
			return
		}
	}
}

// emptyDirectoryAfterPlannedRemovals checks whether rmdir will see an empty
// directory after earlier planned removals and moves have happened.
func (p *applyPreflight) emptyDirectoryAfterPlannedRemovals(path string) (bool, error) {
	path = filepath.Clean(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		if kind, ok := p.paths[path]; os.IsNotExist(err) && ok && kind == preflightDirectory {
			return !p.hasPlannedExistingChild(path), nil
		}
		return false, err
	}
	for _, entry := range entries {
		if kind, ok := p.paths[filepath.Join(path, entry.Name())]; ok && kind == preflightRemoved {
			continue
		}
		return false, nil
	}
	return !p.hasPlannedExistingChild(path), nil
}

func (p *applyPreflight) hasPlannedExistingChild(path string) bool {
	path = filepath.Clean(path)
	for plannedPath, kind := range p.paths {
		if kind == preflightRemoved {
			continue
		}
		if filepath.Dir(plannedPath) == path {
			return true
		}
	}
	return false
}

// hasRemovedAncestor lets planned removals hide all descendants. Without this,
// an action after a planned rmdir could incorrectly validate against stale disk
// contents under a directory that will no longer exist.
func (p *applyPreflight) hasRemovedAncestor(path string) bool {
	for parent := filepath.Dir(filepath.Clean(path)); ; parent = filepath.Dir(parent) {
		if kind, ok := p.paths[parent]; ok && kind == preflightRemoved {
			return true
		}
		if next := filepath.Dir(parent); next == parent {
			return false
		}
	}
}
