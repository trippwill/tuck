package app

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

const version = "dev"

var commandNotFound = func(_ context.Context, cmd *cli.Command, name string) {
	fmt.Fprintf(cmd.Root().ErrWriter, "Incorrect Usage: unknown command %q\n\n", name)
	if cmd == cmd.Root() {
		cli.ShowRootCommandHelpAndExit(cmd, ExitCommandLine)
		return
	}
	cli.ShowSubcommandHelpAndExit(cmd, ExitCommandLine)
}

func rootCommand() *cli.Command {
	return &cli.Command{
		Name:      "tuck",
		Usage:     "manage dotfiles by linking package leaves into a target tree",
		UsageText: "tuck [global-flags] <command> [args]",
		Version:   version,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "root", Usage: "use the root context (target /); default is home"},
			&cli.StringFlag{Name: "source", Aliases: []string{"s"}, Usage: "select active source by enabled id"},
			&cli.BoolFlag{Name: "json", Usage: "machine-readable output"},
			&cli.BoolFlag{Name: "apply", Usage: "execute the plan (mutating verbs plan only without it)"},
			&cli.BoolFlag{Name: "no-color", Usage: "disable colored output (implied by --json)"},
		},
		Commands: []*cli.Command{
			stubCommand("deploy", "create managed links for a package", "<package>..."),
			stubCommand("undeploy", "remove managed links for a package", "<package>..."),
			stubCommand("redeploy", "refresh managed links (undeploy + deploy)", "<package>..."),
			stubCommand("adopt", "move a real file into a package, then link it", "<package> <file>"),
			stubCommand("eject", "remove a managed link, restoring the real file", "<link>"),
			stubCommand("packages", "list the active source's packages", ""),
			stubCommand("tree", "show package contents", "[package]"),
			stubCommand(
				"status",
				"show managed/conflict state",
				"[package] [--path P]",
				&cli.StringFlag{Name: "path", Usage: "show status for a target path"},
			),
			sourceCommand(),
		},
		CommandNotFound: commandNotFound,
	}
}

func sourceCommand() *cli.Command {
	return &cli.Command{
		Name:            "source",
		Usage:           "manage enabled dotfiles sources",
		UsageText:       "tuck source <command> [args]",
		CommandNotFound: commandNotFound,
		Commands: []*cli.Command{
			stubCommand(
				"enable",
				"enable a dotfiles repo on this machine",
				"<path> [--default]",
				&cli.BoolFlag{Name: "default", Usage: "make this the default source"},
			),
			stubCommand("list", "list enabled sources", ""),
		},
	}
}

func stubCommand(name, usage, argsUsage string, flags ...cli.Flag) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: argsUsage,
		Flags:     flags,
		Action: func(context.Context, *cli.Command) error {
			return cli.Exit(fmt.Sprintf("error: command %q is not implemented yet", name), ExitRuntime)
		},
	}
}
