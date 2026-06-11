package app

import (
	"context"

	"github.com/trippwill/tuck/internal/state"
	"github.com/urfave/cli/v3"
)

func sourceCommand() *cli.Command {
	return &cli.Command{
		Name:            "source",
		Usage:           "manage enabled dotfiles sources",
		UsageText:       "tuck source <command> [args]",
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
				Arguments: []cli.Argument{
					requiredStringArgs("id", "<id>"),
				},
				OnUsageError: commandUsageError,
				Action:       sourceRmAction,
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
				Arguments: []cli.Argument{
					requiredStringArgs("id", "<id>"),
				},
				OnUsageError: commandUsageError,
				Action:       sourceDefaultAction,
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

	return renderSourceAdd(cmd, registry, source)
}

func sourceListAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return cli.Exit("error: source list accepts no arguments", ExitFail)
	}
	registry, err := state.Load()
	if err != nil {
		return renderError(cmd, "source list", err)
	}

	return renderSourceList(cmd, registry)
}

func sourceRmAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return cli.Exit("error: source rm accepts exactly one <id>", ExitFail)
	}
	_ = cmd.StringArgs("id")[0]
	return notImplemented("source rm")(ctx, cmd)
}

func sourceDefaultAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return cli.Exit("error: source default accepts exactly one <id>", ExitFail)
	}
	_ = cmd.StringArgs("id")[0]
	return notImplemented("source default")(ctx, cmd)
}
