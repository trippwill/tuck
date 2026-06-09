package plan

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pathutil"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
	"github.com/trippwill/tuck/internal/target"
)

//go:generate go run ../../cmd/errgen -types ErrPlan
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

type Action struct {
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	LinkPath string `json:"linkPath,omitempty"`
	Payload  string `json:"payload,omitempty"`
	Target   string `json:"target,omitempty"`
}

type Conflict struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Package string `json:"package,omitempty"`
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
	targetRoot := options.TargetRoot
	if targetRoot == "" {
		targetRoot = os.Getenv("HOME")
	}
	if targetRoot == "" {
		return UsePlan{}, AppErrMsg(ErrApply, "HOME is not set")
	}
	targetRoot = filepath.Clean(targetRoot)

	registry, err := state.Load()
	if err != nil {
		return UsePlan{}, err
	}
	source, err := resolve.ActiveSource(registry, options.SourceID)
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
				usePlan.Conflicts = append(usePlan.Conflicts, conflict("path_mismatch", targetPath, pkg.Identity.String(), err.Error()))
				blockedRelPrefixes = append(blockedRelPrefixes, entry.Rel)
				continue
			}
			class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
				usePlan.Actions = append(usePlan.Actions, Action{Type: "mkdir", Path: targetPath})
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
				usePlan.Conflicts = append(usePlan.Conflicts, conflict("path_mismatch", targetPath, pkg.Identity.String(), err.Error()))
				continue
			}
			if owner, ok := plannedTargets[targetPath]; ok && owner != pkg.Identity.String()+":"+entry.Rel {
				usePlan.Conflicts = append(usePlan.Conflicts, conflict("multiple_providers", targetPath, pkg.Identity.String(), "multiple packages provide this target"))
				continue
			}
			plannedTargets[targetPath] = pkg.Identity.String() + ":" + entry.Rel

			class := target.Classify(targetPath, source, packages.ContextHome, targetRoot, &pkg.Identity, entry.Rel)
			switch class.Kind {
			case target.Absent:
				payload, err := pathutil.SymlinkPayload(targetPath, entry.Path)
				if err != nil {
					usePlan.Conflicts = append(usePlan.Conflicts, conflict("path_mismatch", targetPath, pkg.Identity.String(), err.Error()))
					continue
				}
				usePlan.Actions = append(usePlan.Actions, Action{Type: "symlink", LinkPath: targetPath, Payload: payload, Target: entry.Path})
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

func blockedByDirectoryConflict(rel string, blockedPrefixes []string) bool {
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

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

func conflict(code, path, pkg, message string) Conflict {
	return Conflict{Code: code, Path: path, Package: pkg, Message: message}
}
