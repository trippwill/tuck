package app

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

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
