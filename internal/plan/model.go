package plan

import "github.com/trippwill/tuck/internal/target"

//go:generate go run ../../cmd/errgen -type ErrPlan
type ErrPlan string

const (
	ErrApply ErrPlan = "plan apply failed"
)

type ActionType string

const (
	ActionMkdir         ActionType = "mkdir"
	ActionRmdir         ActionType = "rmdir"
	ActionSymlink       ActionType = "symlink"
	ActionRemoveSymlink ActionType = "remove_symlink"
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

	physicalPath     string
	physicalLinkPath string
	physicalSrc      string
	physicalDst      string
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

func RemoveSymlinkAction(path string, physicalPath string) Action {
	return Action{Type: ActionRemoveSymlink, Path: path, physicalPath: physicalPath}
}

func MoveAction(src string, physicalSrc string, dst string, physicalDst string) Action {
	return Action{Type: ActionMove, Src: src, physicalSrc: physicalSrc, Dst: dst, physicalDst: physicalDst}
}

func NewConflict(code target.ConflictCode, path, pkg, message string) Conflict {
	return Conflict{Code: code, Path: path, Package: pkg, Message: message}
}
