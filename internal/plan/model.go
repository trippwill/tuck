package plan

import (
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
	"github.com/trippwill/tuck/internal/target"
)

type ErrPlan string

func (e ErrPlan) Error() string { return string(e) }

const (
	ErrApply ErrPlan = "plan apply failed"
)

type ActionType string

const (
	ActionMkdir         ActionType = "mkdir"
	ActionRmdir         ActionType = "rmdir"
	ActionSymlink       ActionType = "symlink"
	ActionCopy          ActionType = "copy"
	ActionPackageConfig ActionType = "package_config"
	ActionRemoveFile    ActionType = "remove_file"
	ActionRemoveSymlink ActionType = "remove_symlink"
	ActionRemoveCopy    ActionType = "remove_copy"
	ActionForgetCopy    ActionType = "forget_copy"
	ActionMove          ActionType = "move"
)

type Action struct {
	Type     ActionType `json:"type"`
	Path     string     `json:"path,omitempty"`
	LinkPath string     `json:"linkPath,omitempty"`
	Payload  string     `json:"payload,omitempty"`
	Target   string     `json:"target,omitempty"`
	Src      string     `json:"src,omitempty"`
	Dst      string     `json:"dst,omitempty"`
	Mode     string     `json:"mode,omitempty"`

	physicalPath     string
	physicalLinkPath string
	physicalSrc      string
	physicalDst      string
	overwrite        bool
	copyRecord       state.Copy
	packageRoot      string
	packageConfig    packages.PackageConfig
}

type Conflict struct {
	Code    target.ConflictCode `json:"code"`
	Path    string              `json:"path"`
	Message string              `json:"message"`
	Package string              `json:"package,omitempty"`
}

type Privilege struct {
	Required  bool   `json:"required"`
	Satisfied *bool  `json:"satisfied,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Plan struct {
	Context   string     `json:"-"`
	DryRun    bool       `json:"dryRun"`
	Applied   bool       `json:"applied"`
	Packages  []string   `json:"packages"`
	Privilege Privilege  `json:"privilege"`
	Actions   []Action   `json:"actions"`
	Conflicts []Conflict `json:"conflicts"`
}

func MkdirAction(path string, physicalPath string) Action {
	return Action{Type: ActionMkdir, Path: path, physicalPath: physicalPath}
}

func RmdirAction(path string, physicalPath string) Action {
	return Action{Type: ActionRmdir, Path: path, physicalPath: physicalPath}
}

func SymlinkAction(linkPath string, physicalLinkPath string, payload string, target string) Action {
	return Action{Type: ActionSymlink, LinkPath: linkPath, physicalLinkPath: physicalLinkPath, Payload: payload, Target: target}
}

func CopyAction(src string, physicalSrc string, dst string, physicalDst string, mode string, overwrite bool, copyRecord state.Copy) Action {
	return Action{Type: ActionCopy, Src: src, physicalSrc: physicalSrc, Dst: dst, physicalDst: physicalDst, Mode: mode, overwrite: overwrite, copyRecord: copyRecord}
}

func PackageConfigAction(path string, packageRoot string, config packages.PackageConfig) Action {
	return Action{Type: ActionPackageConfig, Path: path, packageRoot: packageRoot, packageConfig: config}
}

func RemoveFileAction(path string, physicalPath string) Action {
	return Action{Type: ActionRemoveFile, Path: path, physicalPath: physicalPath}
}

func RemoveSymlinkAction(path string, physicalPath string) Action {
	return Action{Type: ActionRemoveSymlink, Path: path, physicalPath: physicalPath}
}

func RemoveCopyAction(path string, physicalPath string, copyRecord state.Copy) Action {
	return Action{Type: ActionRemoveCopy, Path: path, physicalPath: physicalPath, copyRecord: copyRecord}
}

func ForgetCopyAction(path string, copyRecord state.Copy) Action {
	return Action{Type: ActionForgetCopy, Path: path, copyRecord: copyRecord}
}

func MoveAction(src string, physicalSrc string, dst string, physicalDst string) Action {
	return Action{Type: ActionMove, Src: src, physicalSrc: physicalSrc, Dst: dst, physicalDst: physicalDst}
}

func NewConflict(code target.ConflictCode, path, pkg, message string) Conflict {
	return Conflict{Code: code, Path: path, Package: pkg, Message: message}
}
