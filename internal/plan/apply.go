package plan

import (
	"os"
	"path/filepath"
)

func Apply(usePlan UsePlan) error {
	if len(usePlan.Conflicts) > 0 {
		return AppErrMsg(ErrApply, "cannot apply a plan with conflicts")
	}
	for _, action := range usePlan.Actions {
		switch action.Type {
		case "mkdir":
			if err := os.MkdirAll(action.Path, 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", action.Path)
			}
		case "rmdir":
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
		case "symlink":
			if err := os.MkdirAll(filepath.Dir(action.LinkPath), 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", filepath.Dir(action.LinkPath))
			}
			if err := os.Symlink(action.Payload, action.LinkPath); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create symlink %q", action.LinkPath)
			}
		case "remove_symlink":
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
		case "move":
			if err := os.Rename(action.Src, action.Dst); err != nil {
				return AppErrWrapf(ErrApply, err, "could not move %q to %q", action.Src, action.Dst)
			}
		default:
			return AppErrMsgf(ErrApply, "unknown plan action type %q", action.Type)
		}
	}
	return nil
}
