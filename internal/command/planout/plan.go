package planout

import (
	"fmt"
	"io"
	"strings"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

const Kind output.Kind = "plan"

type Payload struct {
	DryRun    bool       `json:"dryRun"`
	Applied   bool       `json:"applied"`
	Packages  []string   `json:"packages"`
	Privilege Privilege  `json:"privilege"`
	Actions   []Action   `json:"actions"`
	Conflicts []Conflict `json:"conflicts"`
}

type Privilege struct {
	Required  bool   `json:"required"`
	Satisfied *bool  `json:"satisfied,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Action struct {
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	LinkPath string `json:"linkPath,omitempty"`
	Payload  string `json:"payload,omitempty"`
	Target   string `json:"target,omitempty"`
	Src      string `json:"src,omitempty"`
	Dst      string `json:"dst,omitempty"`
}

type Conflict struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Package string `json:"package,omitempty"`
}

func FromPlan(planned plan.Plan) Payload {
	return Payload{
		DryRun:    planned.DryRun,
		Applied:   planned.Applied,
		Packages:  append([]string(nil), planned.Packages...),
		Privilege: fromPrivilege(planned.Privilege),
		Actions:   fromActions(planned.Actions),
		Conflicts: fromConflicts(planned.Conflicts),
	}
}

func (p Payload) Kind() output.Kind {
	return Kind
}

func (p Payload) ExitCode() output.ExitCode {
	if len(p.Conflicts) > 0 {
		return output.ExitFail
	}
	return output.ExitOK
}

func (p Payload) JSONData() any {
	return p
}

func (p Payload) WriteHuman(w io.Writer, inv output.Invocation) error {
	mode := "dry-run"
	if !p.DryRun {
		mode = "apply"
	}
	if _, err := fmt.Fprintf(w, "tuck %s %s   (context: %s, %s)\n\n", inv.Command, packageNames(p.Packages), inv.Context, mode); err != nil {
		return err
	}
	if len(p.Packages) > 0 {
		if _, err := fmt.Fprintf(w, "packages: %s\n\n", strings.Join(p.Packages, " ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "plan:"); err != nil {
		return err
	}
	for _, action := range p.Actions {
		switch action.Type {
		case string(plan.ActionMkdir):
			if _, err := fmt.Fprintf(w, "  + mkdir  %s\n", action.Path); err != nil {
				return err
			}
		case string(plan.ActionRmdir):
			if _, err := fmt.Fprintf(w, "  - rmdir  %s\n", action.Path); err != nil {
				return err
			}
		case string(plan.ActionSymlink):
			if _, err := fmt.Fprintf(w, "  + link   %s -> %s\n", action.LinkPath, action.Target); err != nil {
				return err
			}
		case string(plan.ActionRemoveSymlink):
			if _, err := fmt.Fprintf(w, "  - unlink %s\n", action.Path); err != nil {
				return err
			}
		case string(plan.ActionMove):
			if _, err := fmt.Fprintf(w, "  + move   %s -> %s\n", action.Src, action.Dst); err != nil {
				return err
			}
		}
	}
	if len(p.Conflicts) > 0 {
		if _, err := fmt.Fprintln(w, "\nconflicts:"); err != nil {
			return err
		}
		for _, conflict := range p.Conflicts {
			if _, err := fmt.Fprintf(w, "  ! %s %s", conflict.Code, conflict.Path); err != nil {
				return err
			}
			if conflict.Message != "" {
				if _, err := fmt.Fprintf(w, " (%s)", conflict.Message); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(w, "\n%d actions, %d conflicts\n", len(p.Actions), len(p.Conflicts)); err != nil {
		return err
	}
	if p.DryRun && len(p.Conflicts) == 0 {
		if _, err := fmt.Fprintln(w, "re-run with --apply to execute"); err != nil {
			return err
		}
	}
	return nil
}

func fromPrivilege(privilege plan.Privilege) Privilege {
	return Privilege{
		Required:  privilege.Required,
		Satisfied: privilege.Satisfied,
		Reason:    privilege.Reason,
	}
}

func fromActions(actions []plan.Action) []Action {
	out := make([]Action, 0, len(actions))
	for _, action := range actions {
		out = append(out, Action{
			Type:     string(action.Type),
			Path:     action.Path,
			LinkPath: action.LinkPath,
			Payload:  action.Payload,
			Target:   action.Target,
			Src:      action.Src,
			Dst:      action.Dst,
		})
	}
	return out
}

func fromConflicts(conflicts []plan.Conflict) []Conflict {
	out := make([]Conflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		out = append(out, Conflict{
			Code:    string(conflict.Code),
			Path:    conflict.Path,
			Message: conflict.Message,
			Package: conflict.Package,
		})
	}
	return out
}

func packageNames(identities []string) string {
	names := make([]string, 0, len(identities))
	for _, identity := range identities {
		parts := strings.Split(identity, ":")
		if len(parts) == 3 {
			names = append(names, parts[2])
			continue
		}
		names = append(names, identity)
	}
	return strings.Join(names, " ")
}
