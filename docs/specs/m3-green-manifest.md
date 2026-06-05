# M3 green: repository manifest loader

## Context

M3 implements the first Sources-slice engine story from `docs/backlog.md`: read and validate a repository manifest at `<repo>/tuck.toml`. The red tests are in `internal/manifest/manifest_test.go` and currently fail because the `internal/manifest` package API is not implemented.

The behavior source of truth is `docs/cli-spec.md` §5.2:

- `name` is required and is the short source id.
- `description` is optional.
- Unknown top-level keys, including future reserved sections such as `[security]`, are ignored.
- A missing or unreadable manifest is classified as `manifest_missing` at the CLI layer.
- A malformed manifest or invalid/missing `name` is classified as `manifest_invalid` at the CLI layer.

## Scope

Implement only the pure engine package needed to turn the M3 unit tests green. Do not wire manifest loading into CLI commands, source state, active-source resolution, JSON output, or exit-code mapping in this story.

## Package API

Add `internal/manifest` with this small API:

```go
package manifest

type Manifest struct {
	Name        string
	Description string
}

var ErrMissing error
var ErrInvalid error

func Load(repoRoot string) (Manifest, error)
```

`Load` reads `filepath.Join(repoRoot, "tuck.toml")`, decodes it as TOML, validates it, and returns a `Manifest`.

Errors should wrap `ErrMissing` or `ErrInvalid` so callers can use `errors.Is`. Keep CLI-specific error codes and exit code `3` translation out of this package; later source/CLI stories will map these engine errors to user-facing diagnostics.

## TOML parsing

Add a real TOML parser dependency, preferably `github.com/pelletier/go-toml/v2`, rather than hand-rolling a partial parser.

Decode into an internal struct:

```go
type file struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}
```

Use the parser's default permissive behavior for unknown fields; do not enable unknown-field rejection. This preserves the spec's additive manifest format and allows future sections such as `[security]`.

## Validation rules

After decoding:

- `name` must be present.
- `name` must not be empty or whitespace-only.
- `name` must not contain `/`.
- `name` must not contain `:`.
- `description` may be empty.

Classify read failures as `ErrMissing`, including absent and unreadable `tuck.toml`. Classify TOML decode failures and validation failures as `ErrInvalid`.

## Task breakdown

1. Add the TOML parser dependency
   - Run `go get github.com/pelletier/go-toml/v2@latest`.
   - Confirm `go.mod` and `go.sum` update cleanly.

2. Implement the manifest package
   - Add `internal/manifest/manifest.go`.
   - Define `Manifest`, `ErrMissing`, `ErrInvalid`, and `Load`.
   - Read `<repoRoot>/tuck.toml`; wrap all read errors with `ErrMissing`.
   - Decode TOML into a private file struct; wrap decode errors with `ErrInvalid`.
   - Validate `name`; wrap validation errors with `ErrInvalid`.

3. Keep the package engine-only
   - Do not import `internal/cli`.
   - Do not return process exit codes or user-facing CLI messages from `internal/manifest`.
   - Keep future CLI diagnostics able to recover details from the wrapped error string if needed.

4. Verify green
   - Run `go test ./internal/manifest`.
   - Run `go test ./internal/...`.
   - Run `go build ./...` to ensure production builds work without `tuck_testhooks`.

5. Complete story bookkeeping
   - If the implementation and tests are green, mark M3 complete in `docs/backlog.md`.

## Acceptance criteria

- `go test ./internal/manifest` passes.
- Existing unit tests still pass with `go test ./internal/...`.
- `go build ./...` passes tag-less.
- Unknown TOML keys and sections do not fail manifest loading.
- Invalid manifests can be classified with `errors.Is(err, manifest.ErrInvalid)`.
- Missing or unreadable manifests can be classified with `errors.Is(err, manifest.ErrMissing)`.
