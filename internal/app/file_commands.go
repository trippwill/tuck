package app

import (
	"context"

	statuspkg "github.com/trippwill/tuck/internal/status"
	"github.com/urfave/cli/v3"
)

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
		Action:    statusAction,
	}
}

func statusAction(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("root") {
		return cli.Exit("error: --root is not implemented for status yet", ExitFail)
	}
	if cmd.Args().Len() != 1 {
		return cli.Exit("error: status requires exactly one path", ExitFail)
	}
	result, err := statuspkg.File(cmd.Args().First(), statuspkg.Options{
		SourceID: cmd.String("source"),
	})
	if err != nil {
		return renderError(cmd, "status", err)
	}
	return renderStatus(cmd, result)
}
