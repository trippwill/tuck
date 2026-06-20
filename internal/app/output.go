package app

import (
	"io"
	"os"

	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

func rendererFor(cmd *cli.Command) output.Renderer {
	root := cmd.Root()
	jsonOutput := cmd.Bool("json")
	format := output.Human
	if jsonOutput {
		format = output.JSON
	}
	return output.NewRenderer(output.Options{
		Format: format,
		Color:  colorEnabled(cmd, jsonOutput),
		Out:    root.Writer,
		Err:    root.ErrWriter,
	})
}

func finish(exitCode output.ExitCode, err error) error {
	if err != nil {
		return err
	}
	if exitCode != output.ExitOK {
		return cli.Exit("", int(exitCode))
	}
	return nil
}

func colorEnabled(cmd *cli.Command, jsonOutput bool) bool {
	return !jsonOutput && !cmd.Bool("no-color") && os.Getenv("NO_COLOR") == ""
}

func writeEnvelope(out io.Writer, command string, context string, kind string, data any, exitCode int) error {
	return output.WriteEnvelope(out, output.Command(command), context, output.Kind(kind), data, output.ExitCode(exitCode))
}
