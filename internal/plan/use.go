package plan

import (
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/target"
)

//go:generate go run ../../cmd/errgen -type ErrPlan
type ErrPlan string

const (
	ErrApply             ErrPlan = "plan apply failed"
	ErrPrivilegeRequired ErrPlan = "privilege required"
)

type UseOptions struct {
	Refs       []string
	All        bool
	SourceID   string
	Context    string
	TargetRoot string
	Apply      bool
}

type DropOptions struct {
	Refs       []string
	SourceID   string
	Context    string
	TargetRoot string
	Apply      bool
}

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

type UsePlan struct {
	Command   string     `json:"-"`
	Context   string     `json:"-"`
	DryRun    bool       `json:"dryRun"`
	Applied   bool       `json:"applied"`
	Packages  []string   `json:"packages"`
	Privilege Privilege  `json:"privilege"`
	Actions   []Action   `json:"actions"`
	Conflicts []Conflict `json:"conflicts"`
}

func BuildUse(options UseOptions) (UsePlan, error) {
	context := selectedContext(options.Context)
	scope, err := domain.NewTargetScope(context, options.TargetRoot, true)
	if err != nil {
		return UsePlan{}, AppErrMsg(ErrApply, err.Error())
	}

	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return UsePlan{}, err
	}

	resolvedPackages, err := packages.Resolve(source, scope.Context, options.Refs, options.All)
	if err != nil {
		return UsePlan{}, err
	}

	usePlan := UsePlan{
		Command:   "package use",
		Context:   scope.Context,
		DryRun:    !options.Apply,
		Applied:   false,
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}
	for _, pkg := range resolvedPackages {
		usePlan.Packages = append(usePlan.Packages, pkg.Identity.String())
	}

	plannedTargets := make(map[string]string)
	for _, pkg := range resolvedPackages {
		blockedRelPrefixes := make([]string, 0)
		for _, entry := range packages.Directories(pkg.Entries) {
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, scope.LogicalRoot)
			if err != nil {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := target.ClassifyAt(targetPath, scope.PhysicalPath(targetPath), source, scope.Context, scope.LogicalRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
				usePlan.Actions = append(usePlan.Actions, Action{Type: ActionMkdir, Path: targetPath, physicalPath: scope.PhysicalPath(targetPath)})
			case target.RealDirectory:
			default:
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(class.ConflictCode(), targetPath, pkg.Identity.String(), class.Message))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
			}
		}

		for _, entry := range packages.Leaves(pkg.Entries) {
			if blockedByDirectoryConflict(entry.Rel, blockedRelPrefixes) {
				continue
			}
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, scope.LogicalRoot)
			if err != nil {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetPath]; ok && owner != pkg.Identity.String()+":"+entry.Rel {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictMultipleProviders, targetPath, pkg.Identity.String(), "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetPath] = pkg.Identity.String() + ":" + entry.Rel

			physicalTargetPath := scope.PhysicalPath(targetPath)
			class := target.ClassifyAt(targetPath, physicalTargetPath, source, scope.Context, scope.LogicalRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
				payload, err := pathutil.SymlinkPayload(physicalTargetPath, entry.Path)
				if err != nil {
					usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
					continue
				}
				usePlan.Actions = append(usePlan.Actions, Action{Type: ActionSymlink, LinkPath: targetPath, physicalLinkPath: physicalTargetPath, Payload: payload, Target: entry.Path})
			case target.ManagedSelected:
			default:
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(class.ConflictCode(), targetPath, pkg.Identity.String(), class.Message))
			}
		}
	}

	if len(usePlan.Conflicts) > 0 {
		usePlan.Actions = []Action{}
		return usePlan, nil
	}
	markPrivilege(&usePlan)
	if options.Apply {
		if privilegeDenied(usePlan) {
			return usePlan, AppErrMsg(ErrPrivilegeRequired, "root-context write requires elevated privileges")
		}
		if err := Apply(usePlan); err != nil {
			return usePlan, err
		}

		usePlan.Applied = true
		usePlan.DryRun = false
	}
	return usePlan, nil
}

func BuildDrop(options DropOptions) (UsePlan, error) {
	context := selectedContext(options.Context)
	scope, err := domain.NewTargetScope(context, options.TargetRoot, true)
	if err != nil {
		return UsePlan{}, AppErrMsg(ErrApply, err.Error())
	}

	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return UsePlan{}, err
	}

	resolvedPackages, err := packages.Resolve(source, scope.Context, options.Refs, false)
	if err != nil {
		return UsePlan{}, err
	}

	dropPlan := UsePlan{
		Command:   "package drop",
		Context:   scope.Context,
		DryRun:    !options.Apply,
		Applied:   false,
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}
	for _, pkg := range resolvedPackages {
		dropPlan.Packages = append(dropPlan.Packages, pkg.Identity.String())
	}

	for _, pkg := range resolvedPackages {
		for _, entry := range packages.Leaves(pkg.Entries) {
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, scope.LogicalRoot)
			if err != nil {
				dropPlan.Conflicts = append(dropPlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
				continue
			}

			physicalTargetPath := scope.PhysicalPath(targetPath)
			class := target.ClassifyAt(targetPath, physicalTargetPath, source, scope.Context, scope.LogicalRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
			case target.ManagedSelected:
				dropPlan.Actions = append(dropPlan.Actions, Action{Type: ActionRemoveSymlink, Path: targetPath, physicalPath: physicalTargetPath})
			default:
				dropPlan.Conflicts = append(dropPlan.Conflicts, conflict(class.ConflictCode(), targetPath, pkg.Identity.String(), class.Message))
			}
		}
	}

	if len(dropPlan.Conflicts) > 0 {
		dropPlan.Actions = []Action{}
		return dropPlan, nil
	}
	markPrivilege(&dropPlan)
	if options.Apply {
		if privilegeDenied(dropPlan) {
			return dropPlan, AppErrMsg(ErrPrivilegeRequired, "root-context write requires elevated privileges")
		}
		if err := Apply(dropPlan); err != nil {
			return dropPlan, err
		}

		dropPlan.Applied = true
		dropPlan.DryRun = false
	}
	return dropPlan, nil
}

func selectedContext(context string) string {
	if context == packages.ContextRoot {
		return packages.ContextRoot
	}
	return packages.ContextHome
}

func markPrivilege(usePlan *UsePlan) {
	if usePlan.Context != packages.ContextRoot || len(usePlan.Actions) == 0 {
		return
	}
	satisfied := hasRootPrivilege()
	usePlan.Privilege = Privilege{
		Required:  true,
		Satisfied: &satisfied,
		Reason:    "root-context write",
	}
}

func privilegeDenied(usePlan UsePlan) bool {
	return usePlan.Privilege.Required && usePlan.Privilege.Satisfied != nil && !*usePlan.Privilege.Satisfied
}

func blockedByDirectoryConflict(rel string, blockedPrefixes []string) bool {
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func conflict(code target.ConflictCode, path, pkg, message string) Conflict {
	return Conflict{Code: code, Path: path, Package: pkg, Message: message}
}
