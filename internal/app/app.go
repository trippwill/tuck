package app

import (
	"context"
	"fmt"
	"io"

	"github.com/urfave/cli/v3"
)

const version = "dev"

func newCommand(stdout, stderr io.Writer) *cli.Command {
	cmd := &cli.Command{
		Name:                          "tuck",
		Usage:                         "manage dotfiles by linking package leaves into a target tree",
		UsageText:                     "tuck [global-flags] <command> [args]",
		Version:                       version,
		HideVersion:                   true,
		Writer:                        stdout,
		ErrWriter:                     stderr,
		CustomRootCommandHelpTemplate: rootHelpTemplate,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "root", Usage: "use the root context (target /); default is home"},
			&cli.StringFlag{Name: "source", Aliases: []string{"s"}, Usage: "select active source by enabled id"},
			&cli.BoolFlag{Name: "json", Usage: "machine-readable output"},
			&cli.BoolFlag{Name: "apply", Usage: "execute the plan (mutating verbs plan only without it)"},
			&cli.BoolFlag{Name: "no-color", Usage: "disable colored output (implied by --json)"},
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"V"},
				Usage:   "print version",
				Action: func(_ context.Context, cmd *cli.Command, value bool) error {
					if !value {
						return nil
					}
					fmt.Fprintf(cmd.Root().Writer, "tuck %s\n", version)
					return cli.Exit("", ExitOK)
				},
			},
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
		Action:       rootAction,
		OnUsageError: usageError,
		ExitErrHandler: func(context.Context, *cli.Command, error) {
			// Run returns exit codes itself; never let urfave/cli call os.Exit.
		},
	}
	return cmd
}

func rootAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() > 0 {
		return cli.Exit(fmt.Sprintf("error: unknown command %q", cmd.Args().First()), ExitUsage)
	}
	return cli.ShowRootCommandHelp(cmd)
}

func sourceCommand() *cli.Command {
	return &cli.Command{
		Name:         "source",
		Usage:        "manage enabled dotfiles sources",
		UsageText:    "tuck source <command> [args]",
		OnUsageError: usageError,
		Commands: []*cli.Command{
			stubCommand(
				"enable",
				"enable a dotfiles repo on this machine",
				"<path> [--default]",
				&cli.BoolFlag{Name: "default", Usage: "make this the default source"},
			),
			stubCommand("list", "list enabled sources", ""),
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return cli.Exit(fmt.Sprintf("error: unknown command %q", cmd.Args().First()), ExitUsage)
			}
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

func stubCommand(name, usage, argsUsage string, flags ...cli.Flag) *cli.Command {
	return &cli.Command{
		Name:         name,
		Usage:        usage,
		ArgsUsage:    argsUsage,
		Flags:        flags,
		OnUsageError: usageError,
		Action: func(context.Context, *cli.Command) error {
			return cli.Exit(fmt.Sprintf("error: command %q is not implemented yet", name), ExitRuntime)
		},
	}
}

func usageError(_ context.Context, _ *cli.Command, err error, _ bool) error {
	return cli.Exit(fmt.Sprintf("error: %s", err), ExitUsage)
}

const rootHelpTemplate = `tuck — manage dotfiles by linking package leaves into a target tree

usage:
  tuck [global-flags] <command> [args]

commands:
  deploy    <package>...           create managed links for a package
  undeploy  <package>...           remove managed links for a package
  redeploy  <package>...           refresh managed links (undeploy + deploy)
  adopt     <package> <file>       move a real file into a package, then link it
  eject     <link>                 remove a managed link, restoring the real file
  packages                         list the active source's packages
  tree      [package]              show package contents
  status    [package] [--path P]   show managed/conflict state
  source    enable <path>          enable a dotfiles repo on this machine
  source    list                   list enabled sources

global flags:
      --root                use the root context (target /); default is home
  -s, --source ID           select active source by enabled id
      --json                machine-readable output
      --apply               execute the plan (mutating verbs plan only without it)
      --no-color            disable colored output (implied by --json)
  -V, --version             print version
  -h, --help                show help

run ` + "`" + `tuck <command> --help` + "`" + ` for command-specific help.
`
