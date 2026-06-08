package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
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

func renderUsePlan(cmd *cli.Command, usePlan plan.UsePlan) error {
	exitCode := ExitOK
	if len(usePlan.Conflicts) > 0 {
		exitCode = ExitFail
	}
	if cmd.Bool("json") {
		_ = writeJSON(cmd.Root().Writer, envelope{
			SchemaVersion: 1,
			Command:       usePlan.Command,
			Context:       usePlan.Context,
			Kind:          "plan",
			Data:          usePlan,
			ExitCode:      exitCode,
		})
		if exitCode != ExitOK {
			return cli.Exit("", ExitFail)
		}
		return nil
	}

	mode := "dry-run"
	if usePlan.Applied {
		mode = "apply"
	}
	fmt.Fprintf(cmd.Root().Writer, "tuck package use %s   (context: %s, %s)\n\n", packageNames(usePlan.Packages), usePlan.Context, mode)
	if len(usePlan.Packages) > 0 {
		fmt.Fprintf(cmd.Root().Writer, "packages: %s\n\n", strings.Join(usePlan.Packages, " "))
	}
	fmt.Fprintln(cmd.Root().Writer, "plan:")
	for _, action := range usePlan.Actions {
		switch action.Type {
		case "mkdir":
			fmt.Fprintf(cmd.Root().Writer, "  + mkdir  %s\n", action.Path)
		case "symlink":
			fmt.Fprintf(cmd.Root().Writer, "  + link   %s -> %s\n", action.LinkPath, action.Target)
		}
	}
	if len(usePlan.Conflicts) > 0 {
		fmt.Fprintln(cmd.Root().Writer, "\nconflicts:")
		for _, conflict := range usePlan.Conflicts {
			fmt.Fprintf(cmd.Root().Writer, "  ! %s %s", conflict.Code, conflict.Path)
			if conflict.Message != "" {
				fmt.Fprintf(cmd.Root().Writer, " (%s)", conflict.Message)
			}
			fmt.Fprintln(cmd.Root().Writer)
		}
	}
	fmt.Fprintf(cmd.Root().Writer, "\n%d actions, %d conflicts\n", len(usePlan.Actions), len(usePlan.Conflicts))
	if !usePlan.Applied && len(usePlan.Conflicts) == 0 {
		fmt.Fprintln(cmd.Root().Writer, "re-run with --apply to execute")
	}
	if exitCode != ExitOK {
		return cli.Exit("", ExitFail)
	}
	return nil
}

func packageNames(identities []string) string {
	names := make([]string, 0, len(identities))
	for _, identity := range identities {
		parts := strings.Split(identity, ":")
		if len(parts) == 3 {
			names = append(names, parts[2])
			continue
		}
		names = append(names, identity)
	}
	return strings.Join(names, " ")
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
	case errors.Is(err, pkgref.ErrInvalidRef):
		return errorRecord{
			Code:    "invalid_ref",
			Message: "package reference is invalid",
			Hint:    "pass a plain package name without '/', '..', ':', or a source prefix",
		}
	case errors.Is(err, plan.ErrPackageNotFound):
		return errorRecord{
			Code:    "package_not_found",
			Message: trimSentinel(err, plan.ErrPackageNotFound.Error()),
			Hint:    "run tuck pkg list to see packages in the active source",
		}
	case errors.Is(err, plan.ErrApply):
		return errorRecord{
			Code:    "io_error",
			Message: "could not apply package use plan",
			Hint:    "retry after fixing filesystem permissions or target state",
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

func trimSentinel(err error, sentinel string) string {
	message := err.Error()
	prefix := sentinel + ": "
	if strings.HasPrefix(message, prefix) {
		return strings.TrimPrefix(message, prefix)
	}
	return message
}
