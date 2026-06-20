package planconsole

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/plan"
)

const Kind output.Kind = "plan"

func Result(planned plan.Plan) output.Result {
	return output.Result{
		Kind:          Kind,
		Data:          planned,
		ExitCode:      ExitCode(planned),
		ConsoleString: ConsoleString,
	}
}

func ExitCode(planned plan.Plan) output.ExitCode {
	if len(planned.Conflicts) > 0 || privilegeDenied(planned) {
		return output.ExitFail
	}
	return output.ExitOK
}

func ConsoleString(console output.Console, data any) (string, error) {
	planned, ok := data.(plan.Plan)
	if !ok {
		return "", fmt.Errorf("plan console renderer received %T", data)
	}

	inv := console.Invocation
	var b strings.Builder
	mode := "dry-run"
	if !planned.DryRun {
		mode = "apply"
	}
	if _, err := fmt.Fprintf(&b, "tuck %s %s   (context: %s, %s)\n\n", inv.Command, packageNames(planned.Packages), inv.Context, mode); err != nil {
		return "", err
	}
	if len(planned.Packages) > 0 {
		if _, err := fmt.Fprintf(&b, "packages: %s\n\n", strings.Join(planned.Packages, " ")); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprintln(&b, console.Style(output.StyleAccent, "plan:")); err != nil {
		return "", err
	}
	for _, action := range planned.Actions {
		switch action.Type {
		case plan.ActionMkdir:
			if _, err := fmt.Fprintf(&b, "  %s %s\n", console.Style(output.StyleSuccess, "+ mkdir "), action.Path); err != nil {
				return "", err
			}
		case plan.ActionRmdir:
			if _, err := fmt.Fprintf(&b, "  %s %s\n", console.Style(output.StyleDanger, "- rmdir "), action.Path); err != nil {
				return "", err
			}
		case plan.ActionSymlink:
			if _, err := fmt.Fprintf(&b, "  %s %s -> %s\n", console.Style(output.StyleSuccess, "+ link  "), action.LinkPath, action.Target); err != nil {
				return "", err
			}
		case plan.ActionRemoveSymlink:
			if _, err := fmt.Fprintf(&b, "  %s %s\n", console.Style(output.StyleDanger, "- unlink"), action.Path); err != nil {
				return "", err
			}
		case plan.ActionMove:
			if _, err := fmt.Fprintf(&b, "  %s %s -> %s\n", console.Style(output.StyleSuccess, "+ move  "), action.Src, action.Dst); err != nil {
				return "", err
			}
		}
	}
	if len(planned.Conflicts) > 0 {
		if _, err := fmt.Fprintf(&b, "\n%s\n", console.Style(output.StyleDanger, "conflicts:")); err != nil {
			return "", err
		}
		for _, conflict := range planned.Conflicts {
			if _, err := fmt.Fprintf(&b, "  %s %s %s", console.Style(output.StyleDanger, "!"), console.Style(output.StyleDanger, string(conflict.Code)), conflict.Path); err != nil {
				return "", err
			}
			if conflict.Message != "" {
				if _, err := fmt.Fprintf(&b, " (%s)", conflict.Message); err != nil {
					return "", err
				}
			}
			if _, err := fmt.Fprintln(&b); err != nil {
				return "", err
			}
		}
	}
	if privilegeDenied(planned) {
		reason := planned.Privilege.Reason
		if reason == "" {
			reason = "root-context write"
		}
		if _, err := fmt.Fprintf(&b, "\n%s %s\n", console.Style(output.StyleWarning, "privilege required:"), reason); err != nil {
			return "", err
		}
	}
	if _, err := fmt.Fprintf(&b, "\n%s\n", console.Style(output.StyleMuted, fmt.Sprintf("%d actions, %d conflicts", len(planned.Actions), len(planned.Conflicts)))); err != nil {
		return "", err
	}
	if planned.DryRun && len(planned.Conflicts) == 0 {
		if _, err := fmt.Fprintln(&b, console.Style(output.StyleWarning, "re-run with --apply to execute")); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func privilegeDenied(planned plan.Plan) bool {
	return !planned.DryRun && planned.Privilege.Required && planned.Privilege.Satisfied != nil && !*planned.Privilege.Satisfied
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
