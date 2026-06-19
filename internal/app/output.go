package app

import (
	"io"
	"os"

	"github.com/urfave/cli/v3"
)

type envelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Context       string `json:"context,omitempty"`
	Kind          string `json:"kind"`
	Data          any    `json:"data"`
	ExitCode      int    `json:"exitCode"`
}

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

func colorEnabled(cmd *cli.Command, jsonOutput bool) bool {
	return !jsonOutput && !cmd.Bool("no-color") && os.Getenv("NO_COLOR") == ""
}

func writeEnvelope(out io.Writer, command string, context string, kind string, data any, exitCode int) error {
	return writeJSON(out, envelope{
		SchemaVersion: 1,
		Command:       command,
		Context:       context,
		Kind:          kind,
		Data:          data,
		ExitCode:      exitCode,
	})
}

func (r renderer) writeEnvelope(command string, context string, kind string, data any, exitCode int) error {
	return writeEnvelope(r.out, command, context, kind, data, exitCode)
}
