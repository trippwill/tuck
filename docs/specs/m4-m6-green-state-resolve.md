# M4-M6 green: state loading, validation, and active-source resolution

Status: historical milestone note. It describes the red/green state at the time
M4-M6 were implemented and may mention APIs or failing behavior that have since
been refactored.

## Context

M4-M6 are the next engine-only pieces in Vertical A after the M3 manifest loader:

- **M4** discovers and loads machine-local source state from `${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml`, with `TUCK_TEST_STATE_DIR` support behind the existing `tuck_testhooks` seam.
- **M5** validates machine state.
- **M6** resolves the active source from `--source`, the machine default, or the sole enabled source.

The red tests are in:

- `internal/state/state_test.go`
- `internal/state/state_testhooks_test.go`
- `internal/resolve/resolve_test.go`

The stubs currently compile but fail on not-implemented behavior.

## Scope

Implement only the engine packages needed to turn the red tests green:

- `internal/state`
- `internal/resolve`

Do not wire these packages into `internal/app`, acceptance scripts, JSON output, stderr hints, or error-code mapping. Later stories own those user-facing surfaces.

## State package API

Use the existing red-test API:

```go
type Source struct {
	ID       string
	Path     string
	Enabled  bool
	Manifest manifest.Manifest
}

type Registry struct {
	Sources []Source
	Default string
}

type ErrState string

const ErrInvalid ErrState = "invalid state"

func Load() (Registry, error)
```

`Registry.Default` is the single normalized default source id. Do not put a default boolean on each exported `Source`; default is registry-level state after loading, not a property of individual sources.

Errors should follow the repository convention from `AGENTS.md`: typed const sentinel errors with wrapped causes so `errors.Is(err, state.ErrInvalid)` works.

## State file discovery

`Load` must read the package-local `sourcesFile()` helper, not reconstruct the state path inline. This preserves:

- production lookup: `${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml`
- test override lookup under `-tags tuck_testhooks`: `$TUCK_TEST_STATE_DIR/tuck/sources.toml`

If `sources.toml` is absent, return an empty `Registry` and `nil` error. Other read errors should be classified as invalid state unless a more specific state error is added later.

## TOML decoding

Use the existing TOML parser dependency, `github.com/pelletier/go-toml/v2`.

Decode:

```toml
default = "public"

[[source]]
path = "/repo"
id = "public"
enabled = true
```

Important decoding detail: `enabled` defaults to **true**, so model it as `*bool` (or equivalent) in the internal decode struct. A plain `bool` would incorrectly default omitted values to false.

Suggested internal struct:

```go
type file struct {
	Source []sourceEntry `toml:"source"`
	Default string        `toml:"default"`
}

type sourceEntry struct {
	Path    string `toml:"path"`
	ID      string `toml:"id"`
	Enabled *bool  `toml:"enabled"`
}
```

Do not add a default field to `sourceEntry`. The on-disk default is a top-level registry field (`default = "source-id"`) and should decode directly into the file-level `Default` field before being normalized into `Registry.Default`.

Unknown keys should remain ignored by default.

## Normalization and validation

Load should preserve all decoded entries in `Registry.Sources`, including disabled entries, because `source list` will need them later.

Default handling is separate from source preservation: `Registry.Sources` contains source entries without default flags, while `Registry.Default` is copied from the optional top-level default id.

For each entry:

- `Enabled` is true when `enabled` is omitted or true; false only when explicitly false.
- If enabled, canonicalize `Path` using expand/absolute/clean/evaluate-symlinks behavior sufficient for source roots.
- Load the source manifest with `manifest.Load` for enabled entries and store it in `Source.Manifest`.
- Disabled entries are preserved but do not need path canonicalization or manifest loading for this slice.
- No per-source default marker exists in memory or on disk.

Validation rules:

- Malformed TOML is `ErrInvalid`.
- Enabled source ids must be non-empty, unique among enabled entries, and must not contain `/` or `:`.
- The top-level `default`, when present, must name an enabled source.
- Enabled paths must exist and be canonicalizable.
- Enabled source roots must not overlap. Treat equal roots and nested roots as overlap, but allow sibling prefix paths such as `/tmp/repo` and `/tmp/repo-private`.
- Enabled sources must contain a readable, valid `tuck.toml`; manifest errors should be wrapped/classified as `ErrInvalid`.
- Disabled entries do not participate in enabled-only validations, including duplicate ids, missing paths, and invalid/missing manifests.

## Path handling for this slice

This slice needs only source-root canonicalization and overlap checks. Keep the implementation local and narrow if no shared pathutil helper exists yet:

- expand `~` only if needed by tests or straightforward to support
- make paths absolute
- clean lexical components
- evaluate symlinks for existing source roots
- compare overlap path-segment-aware after canonicalization

Broader target/package path primitives still belong to later path work.

## Resolve package API

Use the existing red-test API:

```go
type ErrSource string

const (
	ErrNoSource      ErrSource = "no source"
	ErrUnknownSource ErrSource = "unknown source"
)

func ActiveSource(registry state.Registry, explicitID string) (state.Source, error)
```

Behavior:

1. If `explicitID` is non-empty, return the enabled source with that id.
2. If `explicitID` names a disabled source or no source, return `ErrUnknownSource`.
3. If no explicit id and `registry.Default` is non-empty, return the enabled source with that id. Do not scan per-source default flags. If the default id is stale or disabled, return `ErrNoSource` for this engine slice; valid loaded registries should prevent that state.
4. If no explicit id and exactly one source is enabled, return it.
5. If no explicit id and zero enabled sources, return `ErrNoSource`.
6. If no explicit id and multiple enabled sources but no default, return `ErrNoSource`.

Keep `ErrNoSource` and `ErrUnknownSource` distinct. The CLI layer maps these to different `error.code` values in the JSON envelope; this package should not know about exit codes or output formatting.

## Task breakdown

1. Implement `internal/state.Load`
   - Decode `sourcesFile()`.
   - Return an empty registry for absent state.
   - Preserve enabled and disabled entries.
   - Normalize omitted `enabled` to true.
   - Copy the optional top-level default source id into `Registry.Default`.

2. Implement state validation and error wrapping
   - Wrap TOML, path, manifest, duplicate/default, overlap, and invalid-id failures with `ErrInvalid`.
   - Add helpers for id validation, enabled filtering, canonicalization, and overlap detection.
   - Ensure disabled entries do not trigger enabled-only validation.

3. Implement `internal/resolve.ActiveSource`
   - Apply explicit id, default id, and sole-enabled fallback in spec order.
   - Return typed const sentinel errors that work with `errors.Is`.

4. Verify green
   - Run `go test ./internal/state ./internal/resolve`.
   - Run `go test -tags tuck_testhooks ./internal/state`.
   - Run `go test ./internal/...`.
   - Run `go build ./...`.

5. Complete backlog bookkeeping
   - If implementation and verification are green, mark M4, M5, and M6 complete in `docs/backlog.md`.

## Acceptance criteria

- The red tests in `internal/state` and `internal/resolve` pass.
- `TUCK_TEST_STATE_DIR` wins over `XDG_STATE_HOME` under `-tags tuck_testhooks`.
- `enabled` omitted means enabled.
- Disabled entries are preserved but do not trigger enabled-only validation.
- `Registry.Default` is copied from the top-level registry default id, and no source entry has any default property in memory or on disk.
- `ActiveSource` returns the correct source or distinct typed sentinel errors.
- `go build ./...` passes without test hooks.
