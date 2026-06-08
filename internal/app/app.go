package app

import (
	"context"
	"fmt"

	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/state"
	"github.com/urfave/cli/v3"
)

const version = "dev"

func domainFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "root", Usage: "use the root context (target /); default is home"},
		&cli.StringFlag{Name: "source", Aliases: []string{"s"}, Usage: "select active source by enabled id"},
	}
}

func mutatingDomainFlags() []cli.Flag {
	return append(domainFlags(), []cli.Flag{
		&cli.BoolFlag{Name: "apply", Usage: "execute the plan instead of just printing it"},
	}...)
}

func commandNotFound(_ context.Context, cmd *cli.Command, name string) {
	fmt.Fprintf(cmd.Root().ErrWriter, "Incorrect Usage: unknown command %q\n\n", name)
	if cmd == cmd.Root() {
		cli.ShowRootCommandHelpAndExit(cmd, ExitFail)
		return
	}
	cli.ShowSubcommandHelpAndExit(cmd, ExitFail)
}

func commandUsageError(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	fmt.Fprintf(cmd.Root().ErrWriter, "Incorrect Usage: %s\n\n", err.Error())
	if !cmd.HideHelp {
		_ = cli.ShowSubcommandHelp(cmd)
	}
	return cli.Exit("", ExitFail)
}

func rootCommand() *cli.Command {
	return &cli.Command{
		Name:            "tuck",
		Usage:           "manage dotfiles by linking package leaves into a target tree",
		UsageText:       "tuck [global-flags] <command> [args]",
		Version:         version,
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "machine-readable output"},
			&cli.BoolFlag{Name: "no-color", Usage: "disable colored output (implied by --json)"},
		},
		Commands: []*cli.Command{
			adoptCommand(),
			ejectCommand(),
			statusCommand(),
			packageCommand(),
			sourceCommand(),
		},
		CommandNotFound: commandNotFound,
	}
}

// File operations (top-level).

func adoptCommand() *cli.Command {
	return &cli.Command{
		Name:      "adopt",
		Usage:     "move a real file into a package, then link it back",
		Category:  "files",
		ArgsUsage: "<file> <package>",
		Flags:     mutatingDomainFlags(),
		Action:    notImplemented("adopt"),
	}
}

func ejectCommand() *cli.Command {
	return &cli.Command{
		Name:      "eject",
		Usage:     "remove a managed link, restoring the real file",
		Category:  "files",
		ArgsUsage: "<file>",
		Flags:     mutatingDomainFlags(),
		Action:    notImplemented("eject"),
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "classify a target path (managed/conflict/absent)",
		Category:  "files",
		ArgsUsage: "<file>",
		Flags:     domainFlags(),
		Action:    notImplemented("status"),
	}
}

// Package operations (grouped).

func packageCommand() *cli.Command {
	return &cli.Command{
		Name:            "package",
		Aliases:         []string{"pkg"},
		Usage:           "manage package symlinks",
		UsageText:       "tuck package <command> [args]",
		Category:        "containers",
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
				Action:  notImplemented("package list"),
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
				Action:    notImplemented("package status"),
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

// Source operations (grouped).

func sourceCommand() *cli.Command {
	return &cli.Command{
		Name:            "source",
		Usage:           "manage enabled dotfiles sources",
		UsageText:       "tuck source <command> [args]",
		Category:        "containers",
		HideHelpCommand: true,
		CommandNotFound: commandNotFound,
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "register a dotfiles repo on this machine",
				ArgsUsage: "<path>",
				Arguments: []cli.Argument{
					&cli.StringArgs{
						Name:      "path",
						UsageText: "<path>",
						Min:       1,
						Max:       1,
					},
				},
				OnUsageError: commandUsageError,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "default", Usage: "make this the default source"},
				},
				Action: sourceAddAction,
			},
			{
				Name:      "rm",
				Usage:     "remove a source from this machine",
				ArgsUsage: "<id>",
				Action:    notImplemented("source rm"),
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list enabled sources",
				Action:  sourceListAction,
			},
			{
				Name:      "default",
				Usage:     "set the default source",
				ArgsUsage: "<id>",
				Action:    notImplemented("source default"),
			},
		},
	}
}

func sourceAddAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return cli.Exit("error: source add accepts exactly one <path>", ExitFail)
	}
	path := cmd.StringArgs("path")[0]

	registry, source, err := state.AddSource(path, cmd.Bool("default"))
	if err != nil {
		return renderError(cmd, "source add", err)
	}

	if cmd.Bool("json") {
		return renderSourcesJSON(cmd, "source add", registry)
	}

	defaultValue := "no"
	if registry.Default == source.ID {
		defaultValue = "yes"
	}
	fmt.Fprintf(cmd.Root().Writer, "added source %s\n", source.ID)
	fmt.Fprintf(cmd.Root().Writer, "path: %s\n", source.Path)
	fmt.Fprintf(cmd.Root().Writer, "default: %s\n", defaultValue)
	return nil
}

func sourceListAction(_ context.Context, cmd *cli.Command) error {
	registry, err := state.Load()
	if err != nil {
		return renderError(cmd, "source list", err)
	}

	if cmd.Bool("json") {
		return renderSourcesJSON(cmd, "source list", registry)
	}

	if len(registry.Sources) == 0 {
		fmt.Fprintln(cmd.Root().Writer, "no sources enabled")
		return nil
	}

	fmt.Fprintf(cmd.Root().Writer, "%-8s %-8s %-8s %s\n", "ID", "DEFAULT", "ENABLED", "PATH")
	for _, source := range registry.Sources {
		defaultValue := "no"
		if registry.Default == source.ID {
			defaultValue = "yes"
		}
		enabledValue := "no"
		if source.Enabled {
			enabledValue = "yes"
		}
		fmt.Fprintf(cmd.Root().Writer, "%-8s %-8s %-8s %s\n", source.ID, defaultValue, enabledValue, source.Path)
	}
	return nil
}

func notImplemented(name string) cli.ActionFunc {
	return func(context.Context, *cli.Command) error {
		return cli.Exit(fmt.Sprintf("error: command %q is not implemented yet", name), ExitFail)
	}
}
