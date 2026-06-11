package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
	statuspkg "github.com/trippwill/tuck/internal/status"
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

type packageListData struct {
	Source   string   `json:"source"`
	Packages []string `json:"packages"`
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

func (r renderer) renderSourcesJSON(command string, registry state.Registry) error {
	return r.writeEnvelope(command, "", "sources", buildSourcesData(registry), ExitOK)
}

func renderPackageList(cmd *cli.Command, listing packages.Listing) error {
	r := newRenderer(cmd)
	data := packageListData{Source: listing.Source, Packages: listing.Packages}
	if r.json {
		return r.writeEnvelope("package list", listing.Context, "packages", data, ExitOK)
	}

	fmt.Fprintf(r.out, "tuck package list   (context: %s, source: %s)\n\n", listing.Context, listing.Source)
	for _, name := range listing.Packages {
		fmt.Fprintln(r.out, name)
	}
	if len(listing.Packages) > 0 {
		fmt.Fprintln(r.out)
	}
	fmt.Fprintf(r.out, "%d %s\n", len(listing.Packages), packageNoun(len(listing.Packages)))
	return nil
}

func packageNoun(count int) string {
	if count == 1 {
		return "package"
	}
	return "packages"
}

func renderUsePlan(cmd *cli.Command, usePlan plan.UsePlan) error {
	return renderPlan(cmd, usePlan)
}

func renderPlan(cmd *cli.Command, usePlan plan.UsePlan) error {
	r := newRenderer(cmd)
	exitCode := ExitOK
	if len(usePlan.Conflicts) > 0 {
		exitCode = ExitFail
	}
	if r.json {
		if err := r.writeEnvelope(usePlan.Command, usePlan.Context, "plan", usePlan, exitCode); err != nil {
			return err
		}
		if exitCode != ExitOK {
			return cli.Exit("", ExitFail)
		}
		return nil
	}

	mode := "dry-run"
	if usePlan.Applied {
		mode = "apply"
	}
	fmt.Fprintf(r.out, "tuck %s %s   (context: %s, %s)\n\n", usePlan.Command, packageNames(usePlan.Packages), usePlan.Context, mode)
	if len(usePlan.Packages) > 0 {
		fmt.Fprintf(r.out, "packages: %s\n\n", strings.Join(usePlan.Packages, " "))
	}
	fmt.Fprintln(r.out, "plan:")
	for _, action := range usePlan.Actions {
		switch action.Type {
		case plan.ActionMkdir:
			fmt.Fprintf(r.out, "  + mkdir  %s\n", action.Path)
		case plan.ActionRmdir:
			fmt.Fprintf(r.out, "  - rmdir  %s\n", action.Path)
		case plan.ActionSymlink:
			fmt.Fprintf(r.out, "  + link   %s -> %s\n", action.LinkPath, action.Target)
		case plan.ActionRemoveSymlink:
			fmt.Fprintf(r.out, "  - unlink %s\n", action.Path)
		case plan.ActionMove:
			fmt.Fprintf(r.out, "  + move   %s -> %s\n", action.Src, action.Dst)
		}
	}
	if len(usePlan.Conflicts) > 0 {
		fmt.Fprintln(r.out, "\nconflicts:")
		for _, conflict := range usePlan.Conflicts {
			fmt.Fprintf(r.out, "  ! %s %s", conflict.Code, conflict.Path)
			if conflict.Message != "" {
				fmt.Fprintf(r.out, " (%s)", conflict.Message)
			}
			fmt.Fprintln(r.out)
		}
	}
	fmt.Fprintf(r.out, "\n%d actions, %d conflicts\n", len(usePlan.Actions), len(usePlan.Conflicts))
	if !usePlan.Applied && len(usePlan.Conflicts) == 0 {
		fmt.Fprintln(r.out, "re-run with --apply to execute")
	}
	if exitCode != ExitOK {
		return cli.Exit("", ExitFail)
	}
	return nil
}

func renderStatus(cmd *cli.Command, result statuspkg.Result) error {
	r := newRenderer(cmd)
	if r.json {
		return r.writeEnvelope(result.Command, result.Context, "status", result, ExitOK)
	}

	fmt.Fprintf(r.out, "tuck %s   (context: %s, source: %s)\n\n", result.Command, result.Context, result.Source)
	for _, entry := range result.Entries {
		fmt.Fprintf(r.out, "%-14s %s", entry.State, entry.TargetPath)
		if entry.Package != "" {
			fmt.Fprintf(r.out, " package=%s", entry.Package)
		}
		if entry.Entry != "" {
			fmt.Fprintf(r.out, " entry=%s", entry.Entry)
		}
		if entry.Owner != "" && entry.Owner != entry.Package {
			fmt.Fprintf(r.out, " owner=%s", entry.Owner)
		}
		if entry.Code != "" {
			fmt.Fprintf(r.out, " code=%s", entry.Code)
		}
		if entry.Message != "" {
			fmt.Fprintf(r.out, " (%s)", entry.Message)
		}
		fmt.Fprintln(r.out)
	}
	fmt.Fprintf(r.out, "\n%d %s\n", len(result.Entries), entryNoun(len(result.Entries)))
	return nil
}

func entryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
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

func renderSourceAdd(cmd *cli.Command, registry state.Registry, source state.Source) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source add", registry)
	}

	defaultValue := "no"
	if registry.Default == source.ID {
		defaultValue = "yes"
	}
	fmt.Fprintf(r.out, "added source %s\n", source.ID)
	fmt.Fprintf(r.out, "path: %s\n", source.Path)
	fmt.Fprintf(r.out, "default: %s\n", defaultValue)
	return nil
}

func renderSourceList(cmd *cli.Command, registry state.Registry) error {
	r := newRenderer(cmd)
	if r.json {
		return r.renderSourcesJSON("source list", registry)
	}

	if len(registry.Sources) == 0 {
		fmt.Fprintln(r.out, "no sources enabled")
		return nil
	}

	fmt.Fprintf(r.out, "%-8s %-8s %-8s %s\n", "ID", "DEFAULT", "ENABLED", "PATH")
	for _, source := range registry.Sources {
		defaultValue := "no"
		if registry.Default == source.ID {
			defaultValue = "yes"
		}
		enabledValue := "no"
		if source.Enabled {
			enabledValue = "yes"
		}
		fmt.Fprintf(r.out, "%-8s %-8s %-8s %s\n", source.ID, defaultValue, enabledValue, source.Path)
	}
	return nil
}

func renderError(cmd *cli.Command, command string, err error) error {
	return newRenderer(cmd).renderError(command, err)
}

func (r renderer) renderError(command string, err error) error {
	appErr := classifyError(err)
	if r.json {
		if err := r.writeEnvelope(command, "", "error", errorData{Error: appErr}, ExitFail); err != nil {
			return err
		}
		return cli.Exit("", ExitFail)
	}

	fmt.Fprintf(r.err, "error: %s\n", appErr.Message)
	fmt.Fprintf(r.err, "hint: %s\n", appErr.Hint)
	return cli.Exit("", ExitFail)
}

func classifyError(err error) errorRecord {
	switch {
	case errors.Is(err, state.ErrSourceRoot):
		return errorRecord{
			Code:    "source_root_missing",
			Message: detailMessage(err, "source root is missing or invalid", state.ErrSourceRoot),
			Hint:    "pass the path to an existing source repository",
		}
	case errors.Is(err, state.ErrInvalid):
		return errorRecord{
			Code:    "state_invalid",
			Message: detailMessage(err, "machine source state is invalid", state.ErrInvalid),
			Hint:    "fix or remove the machine-local sources.toml",
		}
	case errors.Is(err, state.ErrWrite):
		return errorRecord{
			Code:    "io_error",
			Message: detailMessage(err, "could not write machine source state", state.ErrWrite),
			Hint:    "retry after fixing filesystem permissions or disk state",
		}
	case errors.Is(err, manifest.ErrMissing):
		return errorRecord{
			Code:    "manifest_missing",
			Message: detailMessage(err, "source manifest is missing", manifest.ErrMissing),
			Hint:    "create tuck.toml in the source repository with a valid name",
		}
	case errors.Is(err, manifest.ErrInvalid):
		return errorRecord{
			Code:    "manifest_invalid",
			Message: detailMessage(err, "source manifest is invalid", manifest.ErrInvalid),
			Hint:    "fix tuck.toml in the source repository",
		}
	case errors.Is(err, pkgref.ErrInvalidRef):
		return errorRecord{
			Code:    "invalid_ref",
			Message: detailMessage(err, "package reference is invalid", pkgref.ErrInvalidRef),
			Hint:    "pass a plain package name that does not start with '.' and does not contain '/', '..', ':', or a source prefix",
		}
	case errors.Is(err, packages.ErrPackageNotFound):
		return errorRecord{
			Code:    "package_not_found",
			Message: detailMessage(err, packages.ErrPackageNotFound.Error(), packages.ErrPackageNotFound),
			Hint:    "run tuck pkg list to see packages in the active source",
		}
	case errors.Is(err, plan.ErrApply):
		return errorRecord{
			Code:    "io_error",
			Message: detailMessage(err, "could not apply target-tree plan", plan.ErrApply),
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
			Message: detailMessage(err, "runtime error"),
			Hint:    "retry after fixing filesystem permissions or disk state",
		}
	}
}

func detailMessage(err error, fallback string, sentinels ...error) string {
	if err == nil {
		return fallback
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return fallback
	}
	for _, sentinel := range sentinels {
		if sentinel == nil {
			continue
		}
		sentinelText := sentinel.Error()
		if message == sentinelText {
			return fallback
		}
		if after, ok := strings.CutPrefix(message, sentinelText+": "); ok && after != "" {
			return after
		}
	}
	return message
}
