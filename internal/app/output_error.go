package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
	"github.com/urfave/cli/v3"
)

type errorData struct {
	Error errorRecord `json:"error"`
}

type errorRecord struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func renderError(cmd *cli.Command, command string, err error) error {
	return newRenderer(cmd).renderError(command, err)
}

func renderInvalidArgs(cmd *cli.Command, command string, message string, hint string) error {
	return newRenderer(cmd).renderErrorRecord(command, "", errorRecord{
		Code:    "invalid_args",
		Message: message,
		Hint:    hint,
	})
}

func (r renderer) renderError(command string, err error) error {
	return r.renderErrorContext(command, "", err)
}

func (r renderer) renderErrorContext(command string, context string, err error) error {
	return r.renderErrorRecord(command, context, classifyError(err))
}

func (r renderer) renderErrorRecord(command string, context string, appErr errorRecord) error {
	if r.json {
		if err := r.writeEnvelope(command, context, "error", errorData{Error: appErr}, ExitFail); err != nil {
			return err
		}
		return cli.Exit("", ExitFail)
	}

	fmt.Fprintf(r.err, "error: %s\n", appErr.Message)
	fmt.Fprintf(r.err, "code: %s\n", appErr.Code)
	fmt.Fprintf(r.err, "hint: %s\n", appErr.Hint)
	return cli.Exit("", ExitFail)
}

func classifyError(err error) errorRecord {
	var invalidArgs output.InvalidArgsError
	if errors.As(err, &invalidArgs) {
		return errorRecord{
			Code:    "invalid_args",
			Message: invalidArgs.Message,
			Hint:    invalidArgs.Hint,
		}
	}
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
			Hint:    "create " + manifest.ManifestFilename + " in the source repository with a valid name",
		}
	case errors.Is(err, manifest.ErrExists):
		return errorRecord{
			Code:    "manifest_exists",
			Message: detailMessage(err, "source manifest already exists", manifest.ErrExists),
			Hint:    "remove the existing " + manifest.ManifestFilename + " or choose a different source path",
		}
	case errors.Is(err, manifest.ErrInvalid):
		return errorRecord{
			Code:    "manifest_invalid",
			Message: detailMessage(err, "source manifest is invalid", manifest.ErrInvalid),
			Hint:    "fix " + manifest.ManifestFilename + " in the source repository",
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
	case errors.Is(err, plan.ErrPrivilegeRequired):
		return errorRecord{
			Code:    "privilege_required",
			Message: detailMessage(err, "root-context write requires elevated privileges", plan.ErrPrivilegeRequired),
			Hint:    "re-run with elevated privileges or omit --apply to inspect the plan",
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
