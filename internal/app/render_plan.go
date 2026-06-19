package app

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/plan"
	"github.com/urfave/cli/v3"
)

func renderUsePlan(cmd *cli.Command, usePlan plan.UsePlan) error {
	return renderPlan(cmd, usePlan)
}

func renderPlan(cmd *cli.Command, usePlan plan.UsePlan) error {
	r := newRenderer(cmd)
	exitCode := ExitOK
	if len(usePlan.Conflicts) > 0 {
		exitCode = ExitFail
	}
	if r.json {
		if err := r.writeEnvelope(usePlan.Command, usePlan.Context, "plan", usePlan, exitCode); err != nil {
			return err
		}
		if exitCode != ExitOK {
			return cli.Exit("", ExitFail)
		}
		return nil
	}

	mode := "dry-run"
	if !usePlan.DryRun {
		mode = "apply"
	}
	fmt.Fprintf(r.out, "tuck %s %s   (context: %s, %s)\n\n", usePlan.Command, packageNames(usePlan.Packages), usePlan.Context, mode)
	if len(usePlan.Packages) > 0 {
		fmt.Fprintf(r.out, "packages: %s\n\n", strings.Join(usePlan.Packages, " "))
	}
	fmt.Fprintln(r.out, "plan:")
	for _, action := range usePlan.Actions {
		switch action.Type {
		case plan.ActionMkdir:
			fmt.Fprintf(r.out, "  + mkdir  %s\n", action.Path)
		case plan.ActionRmdir:
			fmt.Fprintf(r.out, "  - rmdir  %s\n", action.Path)
		case plan.ActionSymlink:
			fmt.Fprintf(r.out, "  + link   %s -> %s\n", action.LinkPath, action.Target)
		case plan.ActionRemoveSymlink:
			fmt.Fprintf(r.out, "  - unlink %s\n", action.Path)
		case plan.ActionMove:
			fmt.Fprintf(r.out, "  + move   %s -> %s\n", action.Src, action.Dst)
		}
	}
	if len(usePlan.Conflicts) > 0 {
		fmt.Fprintln(r.out, "\nconflicts:")
		for _, conflict := range usePlan.Conflicts {
			fmt.Fprintf(r.out, "  ! %s %s", conflict.Code, conflict.Path)
			if conflict.Message != "" {
				fmt.Fprintf(r.out, " (%s)", conflict.Message)
			}
			fmt.Fprintln(r.out)
		}
	}
	fmt.Fprintf(r.out, "\n%d actions, %d conflicts\n", len(usePlan.Actions), len(usePlan.Conflicts))
	if usePlan.DryRun && len(usePlan.Conflicts) == 0 {
		fmt.Fprintln(r.out, "re-run with --apply to execute")
	}
	if exitCode != ExitOK {
		return cli.Exit("", ExitFail)
	}
	return nil
}

func renderPlanError(cmd *cli.Command, usePlan plan.UsePlan, err error) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderErrorContext(usePlan.Command, usePlan.Context, err)
	}
	if renderErr := renderPlan(cmd, usePlan); renderErr != nil {
		return renderErr
	}
	appErr := classifyError(err)
	fmt.Fprintf(r.err, "error: %s\n", appErr.Message)
	fmt.Fprintf(r.err, "code: %s\n", appErr.Code)
	fmt.Fprintf(r.err, "hint: %s\n", appErr.Hint)
	return cli.Exit("", ExitFail)
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
