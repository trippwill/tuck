package app

import (
	"context"

	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/plan"
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
				Flags:     mutatingDomainFlags(),
				Action:    notImplemented("package drop"),
			},
			{
				Name:      "refresh",
				Usage:     "re-sync managed symlinks (drop + use)",
				ArgsUsage: "<package>...",
				Flags:     mutatingDomainFlags(),
				Action:    notImplemented("package refresh"),
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
				Flags:     domainFlags(),
				Action:    notImplemented("package show"),
			},
			{
				Name:      "status",
				Usage:     "show managed/conflict state for packages",
				ArgsUsage: "[package]",
				Flags:     domainFlags(),
				Action:    packageStatusAction,
			},
		},
	}
}

func packageUseAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for package use yet", ExitFail)
	}
	refs := cmd.Args().Slice()
	if len(refs) == 0 && !cmd.Bool("all") {
		return cli.Exit("error: package use requires one or more package refs or --all", ExitFail)
	}
	if len(refs) > 0 && cmd.Bool("all") {
		return cli.Exit("error: package use accepts package refs or --all, not both", ExitFail)
	}
	usePlan, err := plan.BuildUse(plan.UseOptions{
		Refs:     refs,
		All:      cmd.Bool("all"),
		SourceID: cmd.String("source"),
		Apply:    cmd.Bool("apply"),
	})
	if err != nil {
		return renderError(cmd, "package use", err)
	}
	return renderUsePlan(cmd, usePlan)
}

func packageListAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for package list yet", ExitFail)
	}
	if cmd.Args().Len() != 0 {
		return cli.Exit("error: package list accepts no arguments", ExitFail)
	}
	listing, err := packages.List(packages.ListOptions{
		SourceID: cmd.String("source"),
		Context:  packages.ContextHome,
	})
	if err != nil {
		return renderError(cmd, "package list", err)
	}
	return renderPackageList(cmd, listing)
}

func packageStatusAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for package status yet", ExitFail)
	}
	if cmd.Args().Len() > 1 {
		return cli.Exit("error: package status accepts at most one package ref", ExitFail)
	}
	result, err := statuspkg.Package(cmd.Args().First(), statuspkg.Options{
		SourceID: cmd.String("source"),
	})
	if err != nil {
		return renderError(cmd, "package status", err)
	}
	return renderStatus(cmd, result)
}
