package app

import (
	"io"
	"os"

	"github.com/trippwill/tuck/internal/output"
	"github.com/urfave/cli/v3"
)

type renderer struct {
	out   io.Writer
	err   io.Writer
	json  bool
	color bool
}

func newRenderer(cmd *cli.Command) renderer {
	root := cmd.Root()
	jsonOutput := cmd.Bool("json")
	return renderer{
		out:   root.Writer,
		err:   root.ErrWriter,
		json:  jsonOutput,
		color: colorEnabled(cmd, jsonOutput),
	}
}

func rendererFor(cmd *cli.Command) output.Renderer {
	root := cmd.Root()
	jsonOutput := cmd.Bool("json")
	format := output.Human
	if jsonOutput {
		format = output.JSON
	}
	return output.NewRenderer(output.Options{
		Format:        format,
		Color:         colorEnabled(cmd, jsonOutput),
		Out:           root.Writer,
		Err:           root.ErrWriter,
		ClassifyError: classifyOutputError,
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

func (r renderer) writeEnvelope(command string, context string, kind string, data any, exitCode int) error {
	return writeEnvelope(r.out, command, context, kind, data, exitCode)
}

func classifyOutputError(err error) output.Error {
	appErr := classifyError(err)
	return output.Error{
		Code:    appErr.Code,
		Message: appErr.Message,
		Hint:    appErr.Hint,
	}
}
