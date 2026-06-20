package app

import (
	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

func renderInvalidArgs(cmd *cli.Command, command string, message string, hint string) error {
	exitCode, err := rendererFor(cmd).Render(
		output.Invocation{Command: output.Command(command)},
		output.OK(output.InvalidArgs(message, hint)),
	)
	return finish(exitCode, err)
}
