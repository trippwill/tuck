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
		Usage:     "move a real file into a package, then link it back",
		Category:  "files",
		ArgsUsage: "<file> <package>",
		Arguments: []cli.Argument{
			&cli.StringArgs{
				Name:      "file",
				UsageText: "<file>",
				Min:       1,
				Max:       1,
			},
			&cli.StringArgs{
				Name:      "package",
				UsageText: "<package>",
				Min:       1,
				Max:       1,
			},
		},
		OnUsageError: commandUsageError,
		Flags:        mutatingDomainFlags(),
		Action:       adoptAction,
	}
}

func ejectCommand() *cli.Command {
	return &cli.Command{
		Name:      "eject",
		Usage:     "remove a managed link, restoring the real file",
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
		Flags:        mutatingDomainFlags(),
		Action:       ejectAction,
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "classify a target path (managed/conflict/absent)",
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
		Flags:        domainFlags(),
		Action:       statusAction,
	}
}

func adoptAction(_ context.Context, cmd *cli.Command) error {
	if err := noExtraArgs(cmd, "adopt", "adopt accepts exactly <file> <package>", "pass one target file and one package name"); err != nil {
		return err
	}
	contextName := contextFromFlag(cmd)
	outcome := filecmd.Adopt(filecmd.AdoptRequest{
		File:     cmd.StringArgs("file")[0],
		Ref:      cmd.StringArgs("package")[0],
		SourceID: cmd.String("source"),
		Context:  contextName,
		Apply:    cmd.Bool("apply"),
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
