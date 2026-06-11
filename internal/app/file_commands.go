package app

import (
	"context"

	"github.com/trippwill/tuck/internal/plan"
	statuspkg "github.com/trippwill/tuck/internal/status"
	"github.com/urfave/cli/v3"
)

func adoptCommand() *cli.Command {
	return &cli.Command{
		Name:      "adopt",
		Usage:     "move a real file into a package, then link it back",
		Category:  "files",
		ArgsUsage: "<file> <package>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
			&cli.StringArgs{
				Name:      "package",
				UsageText: "<package>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Flags:        mutatingDomainFlags(),
		Action:       adoptAction,
	}
}

func ejectCommand() *cli.Command {
	return &cli.Command{
		Name:      "eject",
		Usage:     "remove a managed link, restoring the real file",
		Category:  "files",
		ArgsUsage: "<file>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Flags:        mutatingDomainFlags(),
		Action:       ejectAction,
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "classify a target path (managed/conflict/absent)",
		Category:  "files",
		ArgsUsage: "<file>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Flags:        domainFlags(),
		Action:       statusAction,
	}
}

func adoptAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for adopt yet", ExitFail)
	}
	if cmd.Args().Present() {
		return cli.Exit("error: adopt accepts exactly <file> <package>", ExitFail)
	}
	file := cmd.StringArgs("file")[0]
	ref := cmd.StringArgs("package")[0]
	adoptPlan, err := plan.BuildAdopt(plan.AdoptOptions{
		File:     file,
		Ref:      ref,
		SourceID: cmd.String("source"),
		Apply:    cmd.Bool("apply"),
	})
	if err != nil {
		return renderError(cmd, "adopt", err)
	}
	return renderPlan(cmd, adoptPlan)
}

func ejectAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for eject yet", ExitFail)
	}
	if cmd.Args().Present() {
		return cli.Exit("error: eject accepts exactly one <file>", ExitFail)
	}
	file := cmd.StringArgs("file")[0]
	ejectPlan, err := plan.BuildEject(plan.EjectOptions{
		File:     file,
		SourceID: cmd.String("source"),
		Apply:    cmd.Bool("apply"),
	})
	if err != nil {
		return renderError(cmd, "eject", err)
	}
	return renderPlan(cmd, ejectPlan)
}

func statusAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for status yet", ExitFail)
	}
	if cmd.Args().Present() {
		return cli.Exit("error: status accepts exactly one <file>", ExitFail)
	}
	file := cmd.StringArgs("file")[0]
	result, err := statuspkg.File(file, statuspkg.Options{
		SourceID: cmd.String("source"),
	})
	if err != nil {
		return renderError(cmd, "status", err)
	}
	return renderStatus(cmd, result)
}
