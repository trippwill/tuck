package app

import (
	"errors"
	"fmt"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
	"github.com/urfave/cli/v3"
)

type envelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Kind          string `json:"kind"`
	Data          any    `json:"data"`
	ExitCode      int    `json:"exitCode"`
}

type sourcesData struct {
	Sources []sourceRecord `json:"sources"`
}

type sourceRecord struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Enabled     bool   `json:"enabled"`
	Default     bool   `json:"default"`
	Description string `json:"description"`
}

type errorData struct {
	Error errorRecord `json:"error"`
}

type errorRecord struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func renderSourcesJSON(cmd *cli.Command, command string, registry state.Registry) error {
	return writeJSON(cmd.Root().Writer, envelope{
		SchemaVersion: 1,
		Command:       command,
		Kind:          "sources",
		Data:          buildSourcesData(registry),
		ExitCode:      ExitOK,
	})
}

func buildSourcesData(registry state.Registry) sourcesData {
	records := make([]sourceRecord, 0, len(registry.Sources))
	for _, source := range registry.Sources {
		records = append(records, sourceRecord{
			ID:          source.ID,
			Path:        source.Path,
			Enabled:     source.Enabled,
			Default:     registry.Default == source.ID,
			Description: source.Manifest.Description,
		})
	}
	return sourcesData{Sources: records}
}

func renderError(cmd *cli.Command, command string, err error) error {
	appErr := classifyError(err)
	if cmd.Bool("json") {
		_ = writeJSON(cmd.Root().Writer, envelope{
			SchemaVersion: 1,
			Command:       command,
			Kind:          "error",
			Data:          errorData{Error: appErr},
			ExitCode:      ExitFail,
		})
		return cli.Exit("", ExitFail)
	}

	fmt.Fprintf(cmd.Root().ErrWriter, "error: %s\n", appErr.Message)
	fmt.Fprintf(cmd.Root().ErrWriter, "hint: %s\n", appErr.Hint)
	return cli.Exit("", ExitFail)
}

func classifyError(err error) errorRecord {
	switch {
	case errors.Is(err, state.ErrSourceRoot):
		return errorRecord{
			Code:    "source_root_missing",
			Message: "source root is missing or invalid",
			Hint:    "pass the path to an existing source repository",
		}
	case errors.Is(err, state.ErrInvalid):
		return errorRecord{
			Code:    "state_invalid",
			Message: "machine source state is invalid",
			Hint:    "fix or remove the machine-local sources.toml",
		}
	case errors.Is(err, state.ErrWrite):
		return errorRecord{
			Code:    "io_error",
			Message: "could not write machine source state",
			Hint:    "retry after fixing filesystem permissions or disk state",
		}
	case errors.Is(err, manifest.ErrMissing):
		return errorRecord{
			Code:    "manifest_missing",
			Message: "source manifest is missing",
			Hint:    "create tuck.toml in the source repository with a valid name",
		}
	case errors.Is(err, manifest.ErrInvalid):
		return errorRecord{
			Code:    "manifest_invalid",
			Message: "source manifest is invalid",
			Hint:    "fix tuck.toml in the source repository",
		}
	case errors.Is(err, resolve.ErrNoSource):
		return errorRecord{
			Code:    "no_source",
			Message: "no active source is available",
			Hint:    "run tuck source add <path> --default or pass --source <id>",
		}
	case errors.Is(err, resolve.ErrUnknownSource):
		return errorRecord{
			Code:    "unknown_source",
			Message: "source is not enabled or does not exist",
			Hint:    "run tuck source list to see enabled sources",
		}
	default:
		return errorRecord{
			Code:    "io_error",
			Message: "runtime error",
			Hint:    "retry after fixing filesystem permissions or disk state",
		}
	}
}
