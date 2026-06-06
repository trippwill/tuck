package app

import (
	"context"
	"fmt"

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

var commandNotFound = func(_ context.Context, cmd *cli.Command, name string) {
	fmt.Fprintf(cmd.Root().ErrWriter, "Incorrect Usage: unknown command %q\n\n", name)
	if cmd == cmd.Root() {
		cli.ShowRootCommandHelpAndExit(cmd, ExitFail)
		return
	}
	cli.ShowSubcommandHelpAndExit(cmd, ExitFail)
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
				Action: notImplemented("package use"),
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
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "default", Usage: "make this the default source"},
				},
				Action: notImplemented("source add"),
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
				Action:  notImplemented("source list"),
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

func notImplemented(name string) cli.ActionFunc {
	return func(context.Context, *cli.Command) error {
		return cli.Exit(fmt.Sprintf("error: command %q is not implemented yet", name), ExitFail)
	}
}
