package plan

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
)

var renameFile = os.Rename

func Apply(plan Plan) error {
	if len(plan.Conflicts) > 0 {
		return apperr.AppErrMsg(ErrApply, "cannot apply a plan with conflicts")
	}
	// Preflight catches predictable failures before the first mutation. The
	// execution switch still checks every action because the filesystem can
	// change between validation and mutation.
	if err := preflightApply(plan); err != nil {
		return err
	}
	for _, action := range plan.Actions {
		switch action.Type {
		case ActionMkdir:
			path := action.fsPath()
			if err := os.MkdirAll(path, 0o755); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not create directory %q", action.Path)
			}
		case ActionRmdir:
			path := action.fsPath()
			info, err := os.Lstat(path)
			if err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not inspect directory %q", action.Path)
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return apperr.AppErrMsgf(ErrApply, "refusing to remove non-directory %q", action.Path)
			}
			if err := os.Remove(path); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not remove directory %q", action.Path)
			}
		case ActionSymlink:
			linkPath := action.fsLinkPath()
			if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not create directory %q", filepath.Dir(action.LinkPath))
			}
			if err := os.Symlink(action.Payload, linkPath); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not create symlink %q", action.LinkPath)
			}
		case ActionCopy:
			if err := copyAction(action); err != nil {
				return err
			}
		case ActionPackageConfig:
			if err := packages.SaveConfig(action.packageRoot, action.packageConfig); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not write package config %q", action.Path)
			}
		case ActionRemoveFile:
			if err := removeRegularFile(action.fsPath(), action.Path); err != nil {
				return err
			}
		case ActionRemoveSymlink:
			path := action.fsPath()
			info, err := os.Lstat(path)
			if err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not inspect symlink %q", action.Path)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return apperr.AppErrMsgf(ErrApply, "refusing to remove non-symlink %q", action.Path)
			}
			if err := os.Remove(path); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not remove symlink %q", action.Path)
			}
		case ActionRemoveCopy:
			if err := removeCopy(action); err != nil {
				return err
			}
		case ActionForgetCopy:
			if err := state.RemoveCopy(action.copyRecord); err != nil {
				return apperr.AppErrWrapf(ErrApply, err, "could not update copied-file state for %q", action.Path)
			}
		case ActionMove:
			if err := moveAction(action); err != nil {
				return err
			}
		default:
			return apperr.AppErrMsgf(ErrApply, "unknown plan action type %q", action.Type)
		}
	}
	return nil
}

func moveAction(action Action) error {
	if err := renameFile(action.fsSrc(), action.fsDst()); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return apperr.AppErrWrapf(ErrApply, err, "could not move %q to %q", action.Src, action.Dst)
	}
	return copyThenRemove(action)
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
func preflightApply(plan Plan) error {
	preflight := applyPreflight{paths: make(map[string]preflightPathKind)}
	for _, action := range plan.Actions {
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
		return p.validateMkdir(action.fsPath(), action.Path)
	case ActionRmdir:
		return p.validateRmdir(action.fsPath(), action.Path)
	case ActionSymlink:
		return p.validateSymlink(action)
	case ActionCopy:
		return p.validateCopy(action)
	case ActionPackageConfig:
		return nil
	case ActionRemoveFile:
		return p.validateRemoveFile(action.fsPath(), action.Path, false)
	case ActionRemoveSymlink:
		return p.validateRemoveSymlink(action.fsPath(), action.Path)
	case ActionRemoveCopy:
		return p.validateRemoveFile(action.fsPath(), action.Path, true)
	case ActionForgetCopy:
		return nil
	case ActionMove:
		return p.validateMove(action)
	default:
		return apperr.AppErrMsgf(ErrApply, "unknown plan action type %q", action.Type)
	}
}

func (p *applyPreflight) record(action Action) {
	switch action.Type {
	case ActionMkdir:
		p.recordMkdir(action.fsPath())
	case ActionRmdir:
		p.paths[filepath.Clean(action.fsPath())] = preflightRemoved
	case ActionSymlink:
		p.paths[filepath.Clean(action.fsLinkPath())] = preflightSymlink
	case ActionCopy:
		p.paths[filepath.Clean(action.fsDst())] = preflightNonDirectory
	case ActionPackageConfig:
	case ActionRemoveFile:
		p.paths[filepath.Clean(action.fsPath())] = preflightRemoved
	case ActionRemoveSymlink:
		p.paths[filepath.Clean(action.fsPath())] = preflightRemoved
	case ActionRemoveCopy:
		p.paths[filepath.Clean(action.fsPath())] = preflightRemoved
	case ActionForgetCopy:
	case ActionMove:
		p.paths[filepath.Clean(action.fsSrc())] = preflightRemoved
		p.paths[filepath.Clean(action.fsDst())] = preflightNonDirectory
	}
}

func (p *applyPreflight) validateMkdir(path string, displayPath string) error {
	path = filepath.Clean(path)
	for dir := path; ; dir = filepath.Dir(dir) {
		exists, isDir, err := p.directoryState(dir)
		if err != nil {
			return apperr.AppErrWrapf(ErrApply, err, "could not inspect directory %q", displayPath)
		}
		if exists {
			if !isDir {
				return apperr.AppErrMsgf(ErrApply, "could not create directory %q: %q is not a directory", displayPath, displayPath)
			}
			return nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			return nil
		}
	}
}

func (p *applyPreflight) validateRmdir(path string, displayPath string) error {
	path = filepath.Clean(path)
	kind, exists, err := p.pathKind(path)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect directory %q", displayPath)
	}
	if !exists {
		return apperr.AppErrMsgf(ErrApply, "could not inspect directory %q: path does not exist", displayPath)
	}
	if kind != preflightDirectory {
		return apperr.AppErrMsgf(ErrApply, "refusing to remove non-directory %q", displayPath)
	}
	empty, err := p.emptyDirectoryAfterPlannedRemovals(path)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect directory %q", displayPath)
	}
	if !empty {
		return apperr.AppErrMsgf(ErrApply, "could not remove directory %q: directory not empty", displayPath)
	}
	return nil
}

func (p *applyPreflight) validateSymlink(action Action) error {
	linkPath := action.fsLinkPath()
	parent := filepath.Dir(linkPath)
	if err := p.validateMkdir(parent, filepath.Dir(action.LinkPath)); err != nil {
		return err
	}
	_, exists, err := p.pathKind(linkPath)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect symlink %q", action.LinkPath)
	}
	if exists {
		return apperr.AppErrMsgf(ErrApply, "could not create symlink %q: path already exists", action.LinkPath)
	}
	return nil
}

func (p *applyPreflight) validateCopy(action Action) error {
	kind, exists, err := p.pathKind(action.fsSrc())
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect copy source %q", action.Src)
	}
	if !exists {
		return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: source does not exist", action.Src, action.Dst)
	}
	if kind == preflightDirectory {
		return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: source is a directory", action.Src, action.Dst)
	}
	if err := p.validateMkdir(filepath.Dir(action.fsDst()), filepath.Dir(action.Dst)); err != nil {
		return err
	}
	kind, exists, err = p.pathKind(action.fsDst())
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect copy destination %q", action.Dst)
	}
	if exists {
		if action.overwrite {
			if plannedKind, planned := p.paths[filepath.Clean(action.fsDst())]; !planned || plannedKind != preflightNonDirectory {
				info, err := os.Lstat(action.fsDst())
				if err != nil && !os.IsNotExist(err) {
					return apperr.AppErrWrapf(ErrApply, err, "could not inspect copy destination %q", action.Dst)
				}
				if err == nil && !info.Mode().IsRegular() {
					return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: destination is not a regular file", action.Src, action.Dst)
				}
			}
		}
		if kind != preflightNonDirectory {
			return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: destination is not a regular file", action.Src, action.Dst)
		}
		if !action.overwrite {
			return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: destination already exists", action.Src, action.Dst)
		}
	}
	return nil
}

func (p *applyPreflight) validateRemoveFile(path string, displayPath string, allowAbsent bool) error {
	kind, exists, err := p.pathKind(path)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect file %q", displayPath)
	}
	if !exists {
		if allowAbsent {
			return nil
		}
		return apperr.AppErrMsgf(ErrApply, "could not remove file %q: path does not exist", displayPath)
	}
	if kind != preflightNonDirectory {
		return apperr.AppErrMsgf(ErrApply, "refusing to remove non-file %q", displayPath)
	}
	return nil
}

func (p *applyPreflight) validateRemoveSymlink(path string, displayPath string) error {
	path = filepath.Clean(path)
	kind, exists, err := p.pathKind(path)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect symlink %q", displayPath)
	}
	if !exists {
		return apperr.AppErrMsgf(ErrApply, "could not inspect symlink %q: path does not exist", displayPath)
	}
	if kind != preflightSymlink {
		return apperr.AppErrMsgf(ErrApply, "refusing to remove non-symlink %q", displayPath)
	}
	return nil
}

func (p *applyPreflight) validateMove(action Action) error {
	_, exists, err := p.pathKind(action.fsSrc())
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect move source %q", action.Src)
	}
	if !exists {
		return apperr.AppErrMsgf(ErrApply, "could not move %q to %q: source does not exist", action.Src, action.Dst)
	}
	_, exists, err = p.pathKind(action.fsDst())
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect move destination %q", action.Dst)
	}
	if exists {
		return apperr.AppErrMsgf(ErrApply, "could not move %q to %q: destination already exists", action.Src, action.Dst)
	}
	if err := p.validateExistingDirectory(filepath.Dir(action.fsDst()), filepath.Dir(action.Dst)); err != nil {
		return err
	}
	return nil
}

func (p *applyPreflight) validateExistingDirectory(path string, displayPath string) error {
	path = filepath.Clean(path)
	exists, isDir, err := p.directoryState(path)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect directory %q", displayPath)
	}
	if !exists {
		return apperr.AppErrMsgf(ErrApply, "directory %q does not exist", displayPath)
	}
	if !isDir {
		return apperr.AppErrMsgf(ErrApply, "%q is not a directory", displayPath)
	}
	return nil
}

func copyAction(action Action) error {
	src := action.fsSrc()
	dst := action.fsDst()
	input, err := os.Open(src)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not open copy source %q", action.Src)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect copy source %q", action.Src)
	}
	if !info.Mode().IsRegular() {
		return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: source is not a regular file", action.Src, action.Dst)
	}
	if action.overwrite {
		outputInfo, err := os.Lstat(dst)
		if err != nil && !os.IsNotExist(err) {
			return apperr.AppErrWrapf(ErrApply, err, "could not inspect copy destination %q", action.Dst)
		}
		if err == nil {
			if !outputInfo.Mode().IsRegular() {
				return apperr.AppErrMsgf(ErrApply, "could not copy %q to %q: destination is not a regular file", action.Src, action.Dst)
			}
			if os.SameFile(info, outputInfo) {
				return finishCopyAction(action, dst)
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not create directory %q", filepath.Dir(action.Dst))
	}
	flags := os.O_WRONLY | os.O_CREATE
	if action.overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	output, err := os.OpenFile(dst, flags, info.Mode().Perm())
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not create copy destination %q", action.Dst)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return apperr.AppErrWrapf(ErrApply, err, "could not copy %q to %q", action.Src, action.Dst)
	}
	if err := output.Close(); err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not close copy destination %q", action.Dst)
	}
	return finishCopyAction(action, dst)
}

func finishCopyAction(action Action, dst string) error {
	if action.Mode != "" {
		mode, err := strconv.ParseUint(action.Mode, 8, 32)
		if err != nil {
			return apperr.AppErrWrapf(ErrApply, err, "invalid copy mode %q", action.Mode)
		}
		if err := os.Chmod(dst, os.FileMode(mode)); err != nil {
			return apperr.AppErrWrapf(ErrApply, err, "could not set copy mode for %q", action.Dst)
		}
	}
	if action.copyRecord.Target != "" {
		if err := state.UpsertCopy(action.copyRecord); err != nil {
			return apperr.AppErrWrapf(ErrApply, err, "could not update copied-file state for %q", action.Dst)
		}
	}
	return nil
}

func copyThenRemove(action Action) error {
	src := action.fsSrc()
	dst := action.fsDst()
	input, err := os.Open(src)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not open move source %q", action.Src)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect move source %q", action.Src)
	}
	if !info.Mode().IsRegular() {
		return apperr.AppErrMsgf(ErrApply, "could not move %q to %q: source is not a regular file", action.Src, action.Dst)
	}
	output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not create move destination %q", action.Dst)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = os.Remove(dst)
		return apperr.AppErrWrapf(ErrApply, err, "could not move %q to %q", action.Src, action.Dst)
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(dst)
		return apperr.AppErrWrapf(ErrApply, err, "could not close move destination %q", action.Dst)
	}
	if os.Geteuid() == 0 {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			if err := os.Chown(dst, int(stat.Uid), int(stat.Gid)); err != nil {
				_ = os.Remove(dst)
				return apperr.AppErrWrapf(ErrApply, err, "could not set move destination owner %q", action.Dst)
			}
		}
	}
	if err := os.Remove(src); err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not remove move source %q", action.Src)
	}
	return nil
}

func removeRegularFile(path string, displayPath string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect file %q", displayPath)
	}
	if !info.Mode().IsRegular() {
		return apperr.AppErrMsgf(ErrApply, "refusing to remove non-file %q", displayPath)
	}
	if err := os.Remove(path); err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not remove file %q", displayPath)
	}
	return nil
}

func removeCopy(action Action) error {
	if _, err := os.Lstat(action.fsPath()); err == nil {
		if err := removeRegularFile(action.fsPath(), action.Path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return apperr.AppErrWrapf(ErrApply, err, "could not inspect copied target %q", action.Path)
	}
	if err := state.RemoveCopy(action.copyRecord); err != nil {
		return apperr.AppErrWrapf(ErrApply, err, "could not update copied-file state for %q", action.Path)
	}
	return nil
}

func (a Action) fsPath() string {
	if a.physicalPath != "" {
		return a.physicalPath
	}
	return a.Path
}

func (a Action) fsLinkPath() string {
	if a.physicalLinkPath != "" {
		return a.physicalLinkPath
	}
	return a.LinkPath
}

func (a Action) fsSrc() string {
	if a.physicalSrc != "" {
		return a.physicalSrc
	}
	return a.Src
}

func (a Action) fsDst() string {
	if a.physicalDst != "" {
		return a.physicalDst
	}
	return a.Dst
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
