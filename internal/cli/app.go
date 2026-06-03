package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

const version = "dev"

func newCommand(stdout, stderr io.Writer) *urfavecli.Command {
	cmd := &urfavecli.Command{
		Name:                          "tuck",
		Usage:                         "manage dotfiles by linking package leaves into a target tree",
		UsageText:                     "tuck [global-flags] <command> [args]",
		Version:                       version,
		HideVersion:                   true,
		Writer:                        stdout,
		ErrWriter:                     stderr,
		CustomRootCommandHelpTemplate: rootHelpTemplate,
		Flags: []urfavecli.Flag{
			&urfavecli.BoolFlag{Name: "root", Usage: "use the root context (target /); default is home"},
			&urfavecli.StringFlag{Name: "source", Aliases: []string{"s"}, Usage: "select active source by enabled id"},
			&urfavecli.BoolFlag{Name: "json", Usage: "machine-readable output"},
			&urfavecli.BoolFlag{Name: "apply", Usage: "execute the plan (mutating verbs plan only without it)"},
			&urfavecli.BoolFlag{Name: "no-color", Usage: "disable colored output (implied by --json)"},
			&urfavecli.BoolFlag{
				Name:    "version",
				Aliases: []string{"V"},
				Usage:   "print version",
				Action: func(_ context.Context, cmd *urfavecli.Command, value bool) error {
					if !value {
						return nil
					}
					fmt.Fprintf(cmd.Root().Writer, "tuck %s\n", version)
					return urfavecli.Exit("", ExitOK)
				},
			},
		},
		Commands: []*urfavecli.Command{
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
				&urfavecli.StringFlag{Name: "path", Usage: "show status for a target path"},
			),
			sourceCommand(),
		},
		Action:       rootAction,
		OnUsageError: usageError,
		ExitErrHandler: func(context.Context, *urfavecli.Command, error) {
			// Run returns exit codes itself; never let urfave/cli call os.Exit.
		},
	}
	return cmd
}

func rootAction(_ context.Context, cmd *urfavecli.Command) error {
	if cmd.NArg() > 0 {
		return urfavecli.Exit(fmt.Sprintf("error: unknown command %q", cmd.Args().First()), ExitUsage)
	}
	return urfavecli.ShowRootCommandHelp(cmd)
}

func sourceCommand() *urfavecli.Command {
	return &urfavecli.Command{
		Name:         "source",
		Usage:        "manage enabled dotfiles sources",
		UsageText:    "tuck source <command> [args]",
		OnUsageError: usageError,
		Commands: []*urfavecli.Command{
			stubCommand(
				"enable",
				"enable a dotfiles repo on this machine",
				"<path> [--default]",
				&urfavecli.BoolFlag{Name: "default", Usage: "make this the default source"},
			),
			stubCommand("list", "list enabled sources", ""),
		},
		Action: func(_ context.Context, cmd *urfavecli.Command) error {
			if cmd.NArg() > 0 {
				return urfavecli.Exit(fmt.Sprintf("error: unknown command %q", cmd.Args().First()), ExitUsage)
			}
			return urfavecli.ShowSubcommandHelp(cmd)
		},
	}
}

func stubCommand(name, usage, argsUsage string, flags ...urfavecli.Flag) *urfavecli.Command {
	return &urfavecli.Command{
		Name:         name,
		Usage:        usage,
		ArgsUsage:    argsUsage,
		Flags:        flags,
		OnUsageError: usageError,
		Action: func(context.Context, *urfavecli.Command) error {
			return urfavecli.Exit(fmt.Sprintf("error: command %q is not implemented yet", name), ExitRuntime)
		},
	}
}

func usageError(_ context.Context, _ *urfavecli.Command, err error, _ bool) error {
	return urfavecli.Exit(fmt.Sprintf("error: %s", err), ExitUsage)
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
