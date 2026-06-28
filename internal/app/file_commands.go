package app

import (
	"context"

	"github.com/trippwill/tuck/internal/command/filecmd"
	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

func adoptCommand() *cli.Command {
	return &cli.Command{
		Name:      "adopt",
		Usage:     "move a real file into a package, then deploy it back",
		Category:  "files",
		ArgsUsage: "<package> <file>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "package",
				UsageText: "<package>",
				Min:       1,
				Max:       1,
			},
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Flags: append(
			mutatingFlags(),
			&cli.BoolFlag{Name: "copy", Usage: "adopt as a copied deployment"},
			&cli.StringFlag{Name: "mode", Usage: "mode for copied deployment"},
			&cli.BoolFlag{Name: "replace", Usage: "replace an existing package file with the target file"},
		),
		Action: adoptAction,
	}
}

func ejectCommand() *cli.Command {
	return &cli.Command{
		Name:      "eject",
		Usage:     "stop managing a deployed file, leaving a real file in place",
		Category:  "files",
		ArgsUsage: "<file>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Flags:        mutatingFlags(),
		Action:       ejectAction,
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Aliases:   []string{"stat"},
		Usage:     "report deployment state for a target path",
		Category:  "files",
		ArgsUsage: "<file>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Action:       statusAction,
	}
}

func adoptAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "adopt", "adopt accepts exactly <package> <file>", "pass one package name and one target file"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := filecmd.Adopt(filecmd.AdoptRequest{
		Ref:      cmd.StringArgs("package")[0],
		File:     cmd.StringArgs("file")[0],
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
		Copy:     cmd.Bool("copy"),
		Mode:     cmd.String("mode"),
		SetMode:  cmd.IsSet("mode"),
		Replace:  cmd.Bool("replace"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: filecmd.CommandAdopt,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func ejectAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "eject", "eject accepts exactly one <file>", "pass exactly one target file"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := filecmd.Eject(filecmd.EjectRequest{
		File:     cmd.StringArgs("file")[0],
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: filecmd.CommandEject,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}

func statusAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "status", "status accepts exactly one <file>", "pass exactly one target file"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := filecmd.Status(filecmd.StatusRequest{
		File:     cmd.StringArgs("file")[0],
		SourceID: cmd.String("source"),
		Context:  contextName,
	})
	exitCode, err := rendererFor(cmd).Render(output.Invocation{
		Command: filecmd.CommandStatus,
		Context: contextName,
	}, outcome)
	return finish(exitCode, err)
}
