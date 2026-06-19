package app

import (
	"context"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/resolve"
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
					&cli.BoolFlag{Name: "init", Usage: "create a missing source manifest before registering"},
					&cli.StringFlag{Name: "name", Usage: "manifest source id to write with --init"},
					&cli.StringFlag{Name: "description", Usage: "manifest description to write with --init"},
				},
				Action: sourceAddAction,
			},
			{
				Name:      "init",
				Usage:     "create a source manifest without registering it",
				ArgsUsage: "<path>",
				Arguments: []cli.Argument{
					requiredStringArgs("path", "<path>"),
				},
				OnUsageError: commandUsageError,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "manifest source id to write"},
					&cli.StringFlag{Name: "description", Usage: "manifest description to write"},
				},
				Action: sourceInitAction,
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
		return renderInvalidArgs(cmd, "source add", "source add accepts exactly one <path>", "pass exactly one source repository path")
	}
	if !cmd.Bool("init") && cmd.String("name") != "" {
		return renderInvalidArgs(cmd, "source add", "--name requires --init", "add --init when writing a new source manifest")
	}
	if !cmd.Bool("init") && cmd.String("description") != "" {
		return renderInvalidArgs(cmd, "source add", "--description requires --init", "add --init when writing a new source manifest")
	}
	path := cmd.StringArgs("path")[0]

	var (
		registry state.Registry
		source   state.Source
		err      error
	)
	if cmd.Bool("init") {
		registry, source, err = state.AddSourceWithInit(path, cmd.Bool("default"), manifest.InitOptions{
			Name:        cmd.String("name"),
			Description: cmd.String("description"),
		})
	} else {
		registry, source, err = state.AddSource(path, cmd.Bool("default"))
	}
	if err != nil {
		return renderError(cmd, "source add", err)
	}

	return renderSourceAdd(cmd, registry, source)
}

func sourceInitAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return renderInvalidArgs(cmd, "source init", "source init accepts exactly one <path>", "pass exactly one source repository path")
	}
	path := cmd.StringArgs("path")[0]
	initialized, err := manifest.Init(path, manifest.InitOptions{
		Name:        cmd.String("name"),
		Description: cmd.String("description"),
	})
	if err != nil {
		return renderError(cmd, "source init", err)
	}
	return renderSourceInit(cmd, initialized)
}

func sourceListAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return renderInvalidArgs(cmd, "source list", "source list accepts no arguments", "remove the extra argument and retry")
	}
	registry, err := state.Load()
	if err != nil {
		return renderError(cmd, "source list", err)
	}

	return renderSourceList(cmd, registry)
}

func sourceRmAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return renderInvalidArgs(cmd, "source rm", "source rm accepts exactly one <id>", "pass exactly one enabled source id")
	}
	id := cmd.StringArgs("id")[0]
	registry, removed, ok, err := state.RemoveSource(id)
	if err != nil {
		return renderError(cmd, "source rm", err)
	}
	if !ok {
		return renderError(cmd, "source rm", resolve.ErrUnknownSource)
	}
	return renderSourceRm(cmd, registry, removed)
}

func sourceDefaultAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Present() {
		return renderInvalidArgs(cmd, "source default", "source default accepts exactly one <id>", "pass exactly one enabled source id")
	}
	id := cmd.StringArgs("id")[0]
	registry, source, ok, err := state.SetDefault(id)
	if err != nil {
		return renderError(cmd, "source default", err)
	}
	if !ok {
		return renderError(cmd, "source default", resolve.ErrUnknownSource)
	}
	return renderSourceDefault(cmd, registry, source)
}
