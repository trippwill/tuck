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
	ErrApply ErrPlan = "plan apply failed"
)

type UseOptions struct {
	Refs       []string
	All        bool
	SourceID   string
	TargetRoot string
	Apply      bool
}

type DropOptions struct {
	Refs       []string
	SourceID   string
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
}

type Conflict struct {
	Code    target.ConflictCode `json:"code"`
	Path    string              `json:"path"`
	Message string              `json:"message"`
	Package string              `json:"package,omitempty"`
}

type Privilege struct {
	Required bool `json:"required"`
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
	targetRoot, err := domain.TargetRoot(options.TargetRoot, true)
	if err != nil {
		return UsePlan{}, AppErrMsg(ErrApply, err.Error())
	}

	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return UsePlan{}, err
	}

	resolvedPackages, err := packages.Resolve(source, packages.ContextHome, options.Refs, options.All)
	if err != nil {
		return UsePlan{}, err
	}

	usePlan := UsePlan{
		Command:   "package use",
		Context:   packages.ContextHome,
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
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, targetRoot)
			if err != nil {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
				usePlan.Actions = append(usePlan.Actions, Action{Type: ActionMkdir, Path: targetPath})
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
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, targetRoot)
			if err != nil {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetPath]; ok && owner != pkg.Identity.String()+":"+entry.Rel {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictMultipleProviders, targetPath, pkg.Identity.String(), "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetPath] = pkg.Identity.String() + ":" + entry.Rel

			class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
				payload, err := pathutil.SymlinkPayload(targetPath, entry.Path)
				if err != nil {
					usePlan.Conflicts = append(usePlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
					continue
				}
				usePlan.Actions = append(usePlan.Actions, Action{Type: ActionSymlink, LinkPath: targetPath, Payload: payload, Target: entry.Path})
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
	if options.Apply {
		if err := Apply(usePlan); err != nil {
			return usePlan, err
		}

		usePlan.Applied = true
		usePlan.DryRun = false
	}
	return usePlan, nil
}

func BuildDrop(options DropOptions) (UsePlan, error) {
	targetRoot, err := domain.TargetRoot(options.TargetRoot, true)
	if err != nil {
		return UsePlan{}, AppErrMsg(ErrApply, err.Error())
	}

	source, err := domain.ActiveSource(options.SourceID)
	if err != nil {
		return UsePlan{}, err
	}

	resolvedPackages, err := packages.Resolve(source, packages.ContextHome, options.Refs, false)
	if err != nil {
		return UsePlan{}, err
	}

	dropPlan := UsePlan{
		Command:   "package drop",
		Context:   packages.ContextHome,
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
			targetPath, _, err := pathutil.PackageToTarget(pkg.Identity.Root, entry.Path, targetRoot)
			if err != nil {
				dropPlan.Conflicts = append(dropPlan.Conflicts, conflict(target.ConflictPathMismatch, targetPath, pkg.Identity.String(), err.Error()))
				continue
			}

			class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
			case target.ManagedSelected:
				dropPlan.Actions = append(dropPlan.Actions, Action{Type: ActionRemoveSymlink, Path: targetPath})
			default:
				dropPlan.Conflicts = append(dropPlan.Conflicts, conflict(class.ConflictCode(), targetPath, pkg.Identity.String(), class.Message))
			}
		}
	}

	if len(dropPlan.Conflicts) > 0 {
		dropPlan.Actions = []Action{}
		return dropPlan, nil
	}
	if options.Apply {
		if err := Apply(dropPlan); err != nil {
			return dropPlan, err
		}

		dropPlan.Applied = true
		dropPlan.DryRun = false
	}
	return dropPlan, nil
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
