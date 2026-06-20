package command

import (
	"errors"

	"github.com/trippwill/tuck/internal/manifest"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pkgref"
	"github.com/trippwill/tuck/internal/plan"
	"github.com/trippwill/tuck/internal/resolve"
	"github.com/trippwill/tuck/internal/state"
)

func ErrorOutcome(err error) output.Outcome {
	return output.OK(ErrorResult(err))
}

func ErrorResult(err error) output.Result {
	switch {
	case errors.Is(err, state.ErrSourceRoot):
		return output.ErrorResult(
			"source_root_missing",
			output.DetailMessage(err, "source root is missing or invalid", state.ErrSourceRoot),
			"pass the path to an existing source repository",
		)
	case errors.Is(err, state.ErrInvalid):
		return output.ErrorResult(
			"state_invalid",
			output.DetailMessage(err, "machine source state is invalid", state.ErrInvalid),
			"fix or remove the machine-local sources.toml",
		)
	case errors.Is(err, state.ErrWrite):
		return output.ErrorResult(
			"io_error",
			output.DetailMessage(err, "could not write machine source state", state.ErrWrite),
			"retry after fixing filesystem permissions or disk state",
		)
	case errors.Is(err, manifest.ErrMissing):
		return output.ErrorResult(
			"manifest_missing",
			output.DetailMessage(err, "source manifest is missing", manifest.ErrMissing),
			"create "+manifest.ManifestFilename+" in the source repository with a valid name",
		)
	case errors.Is(err, manifest.ErrExists):
		return output.ErrorResult(
			"manifest_exists",
			output.DetailMessage(err, "source manifest already exists", manifest.ErrExists),
			"remove the existing "+manifest.ManifestFilename+" or choose a different source path",
		)
	case errors.Is(err, manifest.ErrInvalid):
		return output.ErrorResult(
			"manifest_invalid",
			output.DetailMessage(err, "source manifest is invalid", manifest.ErrInvalid),
			"fix "+manifest.ManifestFilename+" in the source repository",
		)
	case errors.Is(err, pkgref.ErrInvalidRef):
		return output.ErrorResult(
			"invalid_ref",
			output.DetailMessage(err, "package reference is invalid", pkgref.ErrInvalidRef),
			"pass a plain package name that does not start with '.' and does not contain '/', '..', ':', or a source prefix",
		)
	case errors.Is(err, packages.ErrPackageNotFound):
		return output.ErrorResult(
			"package_not_found",
			output.DetailMessage(err, packages.ErrPackageNotFound.Error(), packages.ErrPackageNotFound),
			"run tuck pkg list to see packages in the active source",
		)
	case errors.Is(err, plan.ErrApply):
		return output.ErrorResult(
			"io_error",
			output.DetailMessage(err, "could not apply target-tree plan", plan.ErrApply),
			"retry after fixing filesystem permissions or target state",
		)
	case errors.Is(err, resolve.ErrNoSource):
		return output.ErrorResult("no_source", "no active source is available", "run tuck source add <path> --default or pass --source <id>")
	case errors.Is(err, resolve.ErrUnknownSource):
		return output.ErrorResult("unknown_source", "source is not enabled or does not exist", "run tuck source list to see enabled sources")
	default:
		return output.RuntimeError(output.DetailMessage(err, "runtime error"))
	}
}
