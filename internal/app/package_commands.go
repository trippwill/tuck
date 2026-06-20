package app

import (
	"context"

	"github.com/trippwill/tuck/internal/command/pkgcmd"
	"github.com/trippwill/tuck/internal/output"
	statuspkg "github.com/trippwill/tuck/internal/status"
	"github.com/urfave/cli/v3"
)

func packageCommand() *cli.Command {
	return &cli.Command{
		Name:            "package",
		Aliases:         []string{"pkg"},
		Usage:           "manage package symlinks",
		UsageText:       "tuck package <command> [args]",
		HideHelpCommand: true,
		CommandNotFound: commandNotFound,
		Commands: []*cli.Command{
			{
				Name:      "use",
				Usage:     "create managed symlinks for packages",
				ArgsUsage: "<package>...",
				Arguments: []cli.Argument{
					variadicStringArgs("package", "<package>...", 0),
				},
				OnUsageError: commandUsageError,
				Flags: append(
					mutatingDomainFlags(),
					&cli.BoolFlag{Name: "all", Usage: "activate all packages in the active source"},
				),
				Action: packageUseAction,
			},
			{
				Name:      "drop",
				Usage:     "remove managed symlinks for packages",
				ArgsUsage: "<package>...",
				Arguments: []cli.Argument{
					variadicStringArgs("package", "<package>...", 1),
				},
				OnUsageError: commandUsageError,
				Flags:        mutatingDomainFlags(),
				Action:       packageDropAction,
			},
			{
				Name:      "refresh",
				Usage:     "re-sync managed symlinks (drop + use)",
				ArgsUsage: "<package>...",
				Arguments: []cli.Argument{
					variadicStringArgs("package", "<package>...", 1),
				},
				OnUsageError: commandUsageError,
				Flags:        mutatingDomainFlags(),
				Action:       packageRefreshAction,
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list packages in the active source",
				Flags:   domainFlags(),
				Action:  packageListAction,
			},
			{
				Name:      "show",
				Aliases:   []string{"tree"},
				Usage:     "show package contents",
				ArgsUsage: "<package>",
				Arguments: []cli.Argument{
					requiredStringArgs("package", "<package>"),
				},
				OnUsageError: commandUsageError,
				Flags:        domainFlags(),
				Action:       packageShowAction,
			},
			{
				Name:      "status",
				Usage:     "show managed/conflict state for packages",
				ArgsUsage: "[package]",
				Arguments: []cli.Argument{
					optionalStringArgs("package", "[package]"),
				},
				OnUsageError: commandUsageError,
				Flags:        domainFlags(),
				Action:       packageStatusAction,
			},
		},
	}
}

func packageUseAction(_ context.Context, cmd *cli.Command) error {
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Use(pkgcmd.UseRequest{
		Refs:     cmd.StringArgs("package"),
		All:      cmd.Bool("all"),
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandUse,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageDropAction(_ context.Context, cmd *cli.Command) error {
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Drop(pkgcmd.DropRequest{
		Refs:     cmd.StringArgs("package"),
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandDrop,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageRefreshAction(_ context.Context, cmd *cli.Command) error {
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Refresh(pkgcmd.RefreshRequest{
		Refs:     cmd.StringArgs("package"),
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandRefresh,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageListAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "package list", "package list accepts no arguments", "remove the extra argument and retry"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.List(pkgcmd.ListRequest{
		SourceID: cmd.String("source"),
		Context:  contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandList,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func packageStatusAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "package status", "package status accepts at most one package ref", "remove the extra package ref and retry"); err != nil {
		return err
	}
	ref := ""
	if refs := cmd.StringArgs("package"); len(refs) > 0 {
		ref = refs[0]
	}
	result, err := statuspkg.Package(ref, statuspkg.Options{
		SourceID: cmd.String("source"),
		Context:  contextFromFlag(cmd),
	})
	if err != nil {
		return renderError(cmd, "package status", err)
	}
	return renderStatus(cmd, result)
}

func packageShowAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "package show", "package show accepts exactly one package ref", "pass exactly one package name"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := pkgcmd.Show(pkgcmd.ShowRequest{
		SourceID: cmd.String("source"),
		Context:  contextName,
		Ref:      cmd.StringArgs("package")[0],
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: pkgcmd.CommandShow,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}
