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
		case "symlink":
			if err := os.MkdirAll(filepath.Dir(action.LinkPath), 0o755); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create directory %q", filepath.Dir(action.LinkPath))
			}
			if err := os.Symlink(action.Payload, action.LinkPath); err != nil {
				return AppErrWrapf(ErrApply, err, "could not create symlink %q", action.LinkPath)
			}
		}
	}
	return nil
}
