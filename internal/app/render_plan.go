package app

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/plan"
	"github.com/urfave/cli/v3"
)

func renderPlan(cmd *cli.Command, planned plan.Plan) error {
	r := newRenderer(cmd)
	exitCode := ExitOK
	if len(planned.Conflicts) > 0 {
		exitCode = ExitFail
	}
	if r.json {
		if err := r.writeEnvelope(planned.Command, planned.Context, "plan", planned, exitCode); err != nil {
			return err
		}
		if exitCode != ExitOK {
			return cli.Exit("", ExitFail)
		}
		return nil
	}

	mode := "dry-run"
	if !planned.DryRun {
		mode = "apply"
	}
	fmt.Fprintf(r.out, "tuck %s %s   (context: %s, %s)\n\n", planned.Command, packageNames(planned.Packages), planned.Context, mode)
	if len(planned.Packages) > 0 {
		fmt.Fprintf(r.out, "packages: %s\n\n", strings.Join(planned.Packages, " "))
	}
	fmt.Fprintln(r.out, "plan:")
	for _, action := range planned.Actions {
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
	if len(planned.Conflicts) > 0 {
		fmt.Fprintln(r.out, "\nconflicts:")
		for _, conflict := range planned.Conflicts {
			fmt.Fprintf(r.out, "  ! %s %s", conflict.Code, conflict.Path)
			if conflict.Message != "" {
				fmt.Fprintf(r.out, " (%s)", conflict.Message)
			}
			fmt.Fprintln(r.out)
		}
	}
	fmt.Fprintf(r.out, "\n%d actions, %d conflicts\n", len(planned.Actions), len(planned.Conflicts))
	if planned.DryRun && len(planned.Conflicts) == 0 {
		fmt.Fprintln(r.out, "re-run with --apply to execute")
	}
	if exitCode != ExitOK {
		return cli.Exit("", ExitFail)
	}
	return nil
}

func renderPlanError(cmd *cli.Command, planned plan.Plan, err error) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderErrorContext(planned.Command, planned.Context, err)
	}
	if renderErr := renderPlan(cmd, planned); renderErr != nil {
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
