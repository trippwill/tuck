package plan

import (
	"path/filepath"
	"strings"

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

type RefreshOptions struct {
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

type Plan struct {
	Command   string     `json:"-"`
	Context   string     `json:"-"`
	DryRun    bool       `json:"dryRun"`
	Applied   bool       `json:"applied"`
	Packages  []string   `json:"packages"`
	Privilege Privilege  `json:"privilege"`
	Actions   []Action   `json:"actions"`
	Conflicts []Conflict `json:"conflicts"`
}

func BuildUse(options UseOptions) (Plan, error) {
	op, err := newOperation(operationOptions{
		Command:    "package use",
		Context:    options.Context,
		TargetRoot: options.TargetRoot,
		SourceID:   options.SourceID,
		Apply:      options.Apply,
	})
	if err != nil {
		return Plan{}, err
	}

	resolvedPackages, err := op.resolvePackages(options.Refs, options.All)
	if err != nil {
		return Plan{}, err
	}

	plannedTargets := make(map[string]string)
	for _, pkg := range resolvedPackages {
		blockedRelPrefixes := make([]string, 0)
		for _, entry := range packages.Directories(pkg.Entries) {
			targetEntry, err := target.NewPackageEntry(pkg, entry, op.scope.LogicalRoot, op.scope.PhysicalPath)
			if err != nil {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := targetEntry.Classify(op.source, op.scope.Context, op.scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				op.plan.Actions = append(op.plan.Actions, Action{Type: ActionMkdir, Path: targetEntry.TargetPath, physicalPath: targetEntry.PhysicalPath})
			case target.RealDirectory:
			default:
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
			}
		}

		for _, entry := range packages.Leaves(pkg.Entries) {
			if blockedByDirectoryConflict(entry.Rel, blockedRelPrefixes) {
				continue
			}
			targetEntry, err := target.NewPackageEntry(pkg, entry, op.scope.LogicalRoot, op.scope.PhysicalPath)
			if err != nil {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetEntry.TargetPath]; ok && owner != targetEntry.ProviderKey {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictMultipleProviders, targetEntry.TargetPath, targetEntry.PackageID, "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetEntry.TargetPath] = targetEntry.ProviderKey

			class := targetEntry.Classify(op.source, op.scope.Context, op.scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				payload, err := pathutil.SymlinkPayload(targetEntry.PhysicalPath, targetEntry.Entry.Path)
				if err != nil {
					op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
					continue
				}
				op.plan.Actions = append(op.plan.Actions, Action{Type: ActionSymlink, LinkPath: targetEntry.TargetPath, physicalLinkPath: targetEntry.PhysicalPath, Payload: payload, Target: targetEntry.Entry.Path})
			case target.ManagedSelected:
			default:
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}

	return op.finalize()
}

func BuildDrop(options DropOptions) (Plan, error) {
	op, err := newOperation(operationOptions{
		Command:    "package drop",
		Context:    options.Context,
		TargetRoot: options.TargetRoot,
		SourceID:   options.SourceID,
		Apply:      options.Apply,
	})
	if err != nil {
		return Plan{}, err
	}

	resolvedPackages, err := op.resolvePackages(options.Refs, false)
	if err != nil {
		return Plan{}, err
	}

	for _, pkg := range resolvedPackages {
		for _, entry := range packages.Leaves(pkg.Entries) {
			targetEntry, err := target.NewPackageEntry(pkg, entry, op.scope.LogicalRoot, op.scope.PhysicalPath)
			if err != nil {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				continue
			}

			class := targetEntry.Classify(op.source, op.scope.Context, op.scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
			case target.ManagedSelected:
				op.plan.Actions = append(op.plan.Actions, Action{Type: ActionRemoveSymlink, Path: targetEntry.TargetPath, physicalPath: targetEntry.PhysicalPath})
			default:
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}

	return op.finalize()
}

func BuildRefresh(options RefreshOptions) (Plan, error) {
	op, err := newOperation(operationOptions{
		Command:    "package refresh",
		Context:    options.Context,
		TargetRoot: options.TargetRoot,
		SourceID:   options.SourceID,
		Apply:      options.Apply,
	})
	if err != nil {
		return Plan{}, err
	}

	resolvedPackages, err := op.resolvePackages(options.Refs, false)
	if err != nil {
		return Plan{}, err
	}

	removeActions := make([]Action, 0)
	createActions := make([]Action, 0)
	plannedTargets := make(map[string]string)
	for _, pkg := range resolvedPackages {
		blockedRelPrefixes := make([]string, 0)
		for _, entry := range packages.Directories(pkg.Entries) {
			targetEntry, err := target.NewPackageEntry(pkg, entry, op.scope.LogicalRoot, op.scope.PhysicalPath)
			if err != nil {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := targetEntry.Classify(op.source, op.scope.Context, op.scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				createActions = append(createActions, Action{Type: ActionMkdir, Path: targetEntry.TargetPath, physicalPath: targetEntry.PhysicalPath})
			case target.RealDirectory:
			default:
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
			}
		}

		for _, entry := range packages.Leaves(pkg.Entries) {
			if blockedByDirectoryConflict(entry.Rel, blockedRelPrefixes) {
				continue
			}
			targetEntry, err := target.NewPackageEntry(pkg, entry, op.scope.LogicalRoot, op.scope.PhysicalPath)
			if err != nil {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictPathMismatch, targetEntry.TargetPath, targetEntry.PackageID, err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetEntry.TargetPath]; ok && owner != targetEntry.ProviderKey {
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(target.ConflictMultipleProviders, targetEntry.TargetPath, targetEntry.PackageID, "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetEntry.TargetPath] = targetEntry.ProviderKey

			class := targetEntry.Classify(op.source, op.scope.Context, op.scope.LogicalRoot)
			switch class.Kind {
			case target.Absent:
				action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, &op.plan)
				if ok {
					createActions = append(createActions, action)
				}
			case target.ManagedSelected:
				removeActions = append(removeActions, Action{Type: ActionRemoveSymlink, Path: targetEntry.TargetPath, physicalPath: targetEntry.PhysicalPath})
				action, ok := symlinkAction(targetEntry.TargetPath, targetEntry.PhysicalPath, targetEntry.Entry.Path, targetEntry.PackageID, &op.plan)
				if ok {
					createActions = append(createActions, action)
				}
			default:
				op.plan.Conflicts = append(op.plan.Conflicts, conflict(class.ConflictCode(), targetEntry.TargetPath, targetEntry.PackageID, class.Message))
			}
		}
	}

	if len(op.plan.Conflicts) == 0 {
		op.plan.Actions = append(op.plan.Actions, removeActions...)
		op.plan.Actions = append(op.plan.Actions, createActions...)
	}
	return op.finalize()
}

func symlinkAction(targetPath string, physicalTargetPath string, entryPath string, packageID string, planned *Plan) (Action, bool) {
	payload, err := pathutil.SymlinkPayload(physicalTargetPath, entryPath)
	if err != nil {
		planned.Conflicts = append(planned.Conflicts, conflict(target.ConflictPathMismatch, targetPath, packageID, err.Error()))
		return Action{}, false
	}
	return Action{Type: ActionSymlink, LinkPath: targetPath, physicalLinkPath: physicalTargetPath, Payload: payload, Target: entryPath}, true
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
