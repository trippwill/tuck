package app

import (
	"context"

	"github.com/trippwill/tuck/internal/command/sourcecmd"
	"github.com/trippwill/tuck/internal/output"
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
				Name:      "remove",
				Aliases:   []string{"rm"},
				Usage:     "remove a source from this machine",
				ArgsUsage: "<id>",
				Arguments: []cli.Argument{
					requiredStringArgs("id", "<id>"),
				},
				OnUsageError:  commandUsageError,
				ShellComplete: completeAllSources,
				Action:        sourceRmAction,
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
				OnUsageError:  commandUsageError,
				ShellComplete: completeEnabledSources,
				Action:        sourceDefaultAction,
			},
		},
	}
}

func sourceAddAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "source add", "source add accepts exactly one <path>", "pass exactly one source repository path"); err != nil {
		return err
	}
	outcome := sourcecmd.Add(sourcecmd.AddRequest{
		Path:        cmd.StringArgs("path")[0],
		Default:     cmd.Bool("default"),
		Init:        cmd.Bool("init"),
		Name:        cmd.String("name"),
		Description: cmd.String("description"),
	})
	outcome = output.WithWarnings(outcome, ignoredDomainSelectionWarnings(cmd, sourcecmd.CommandAdd))
	exitCode, err := rendererFor(cmd).Render(output.Invocation{Command: sourcecmd.CommandAdd}, outcome)
	return finish(exitCode, err)
}

func sourceInitAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "source init", "source init accepts exactly one <path>", "pass exactly one source repository path"); err != nil {
		return err
	}
	outcome := sourcecmd.Init(sourcecmd.InitRequest{
		Path:        cmd.StringArgs("path")[0],
		Name:        cmd.String("name"),
		Description: cmd.String("description"),
	})
	outcome = output.WithWarnings(outcome, ignoredDomainSelectionWarnings(cmd, sourcecmd.CommandInit))
	exitCode, err := rendererFor(cmd).Render(output.Invocation{Command: sourcecmd.CommandInit}, outcome)
	return finish(exitCode, err)
}

func sourceListAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "source list", "source list accepts no arguments", "remove the extra argument and retry"); err != nil {
		return err
	}
	outcome := sourcecmd.List()
	outcome = output.WithWarnings(outcome, ignoredDomainSelectionWarnings(cmd, sourcecmd.CommandList))
	exitCode, err := rendererFor(cmd).Render(output.Invocation{Command: sourcecmd.CommandList}, outcome)
	return finish(exitCode, err)
}

func sourceRmAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "source rm", "source rm accepts exactly one <id>", "pass exactly one enabled source id"); err != nil {
		return err
	}
	outcome := sourcecmd.Rm(sourcecmd.IDRequest{ID: cmd.StringArgs("id")[0]})
	outcome = output.WithWarnings(outcome, ignoredDomainSelectionWarnings(cmd, sourcecmd.CommandRm))
	exitCode, err := rendererFor(cmd).Render(output.Invocation{Command: sourcecmd.CommandRm}, outcome)
	return finish(exitCode, err)
}

func sourceDefaultAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "source default", "source default accepts exactly one <id>", "pass exactly one enabled source id"); err != nil {
		return err
	}
	outcome := sourcecmd.Default(sourcecmd.IDRequest{ID: cmd.StringArgs("id")[0]})
	outcome = output.WithWarnings(outcome, ignoredDomainSelectionWarnings(cmd, sourcecmd.CommandDefault))
	exitCode, err := rendererFor(cmd).Render(output.Invocation{Command: sourcecmd.CommandDefault}, outcome)
	return finish(exitCode, err)
}
