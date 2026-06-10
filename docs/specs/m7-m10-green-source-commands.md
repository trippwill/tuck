# M7-M10 green: source command output, errors, add, and list

Status: historical milestone note. It describes the red/green state at the time
M7-M10 were implemented and may mention APIs or failing behavior that have since
been refactored.

## Context

M7-M10 finish Vertical A's first user-facing slice after the M3-M6 source engine:

- **M7** returns meaningful stable error codes in JSON envelopes and stderr.
- **M8** prints actionable human diagnostics and hints on stderr.
- **M9** implements `source add <path> [--default]`.
- **M10** implements `source list` / `source ls`.

This is the first slice that should turn source management into real CLI behavior.
It should add red acceptance coverage before implementation, then build only the
engine/output/application wiring needed to make those scripts pass.

The original red-test phase had to prove two things before implementation:

1. the new tests **compiled** with the then-current codebase; and
2. the source-suite tests **failed for the expected behavioral reason** (for
   example, `source add` / `source list` returning "not implemented" or
   producing the wrong output), not because of a malformed script, missing helper,
   broken harness, or compile error.

## Scope

Implement the source-management vertical slice only:

- error rendering and JSON envelope infrastructure needed by this slice;
- machine-state write/update support needed by `source add`;
- `tuck source add`;
- `tuck source list` and alias `tuck source ls`;
- source-suite acceptance scripts.

Do not implement `source rm`, `source default`, package commands, target-tree
planning, root-context behavior, copy/checksum state, color, or broad JSON
goldens for every command kind. Later stories own those surfaces.

## Historical baseline

- `internal/app` owned the CLI skeleton and command actions. Source subcommands
  returned `not implemented`.
- `internal/manifest.Load` reads and validates `<repo>/tuck.toml`.
- `internal/state.Load` reads the testhook-aware `sources.toml`, preserves
  disabled entries, validates enabled entries, loads manifests, and returns
  `state.Registry`.
- `internal/resolve.ActiveSource` resolves active sources for future domain
  commands, but source add/list do not require an active source.
- The acceptance harness has only the `foundation` suite. `wanthome` should be
  updated before new source scripts depend on it: generated state must use the
  top-level `default = "public"` field, not a per-source `default = true`.

## Output contract for this slice

Follow `docs/cli-spec.md`:

- success primary output goes to stdout;
- human diagnostics and hints go to stderr;
- with `--json`, stdout contains exactly one JSON envelope and stderr is empty;
- process exit codes remain binary: `0` success, `1` failure;
- `schemaVersion` is `1`;
- source commands omit `context`.

### Human success output

Keep human output deterministic and intentionally plain. Acceptance tests should
pin these source-command outputs exactly.

`source add` success:

```text
added source public
path: /home/me/.dotfiles
default: yes
```

Use `default: no` when `--default` was not passed and the added source is not the
registry default.

`source list` with entries:

```text
ID       DEFAULT  ENABLED  PATH
public   yes      yes      /home/me/.dotfiles
private  no       yes      /home/me/.dotfiles-private
```

Rules:

- include all recorded source entries, including disabled entries;
- `DEFAULT` is `yes` only when `Registry.Default == Source.ID`;
- `ENABLED` is `yes` or `no`;
- preserve registry order;
- use stable column padding in tests, or choose a tabwriter but assert the final
  expanded text.

`source list` when no entries are recorded:

```text
no sources enabled
```

This message also covers absent state. It exits `0`.

### JSON success output

Use `kind: "sources"` for both `source add` and `source list`:

```json
{
  "schemaVersion": 1,
  "command": "source list",
  "kind": "sources",
  "data": {
    "sources": [
      {
        "id": "public",
        "path": "/home/me/.dotfiles",
        "enabled": true,
        "default": true,
        "description": "public dotfiles"
      }
    ]
  },
  "exitCode": 0
}
```

Notes:

- `command` is the canonical command name, so `source ls --json` still emits
  `"command": "source list"`.
- Include `description` from the loaded manifest for enabled sources when
  available. Omit it or use `""` for disabled entries whose manifest was not
  loaded; keep one consistent shape in tests.
- Preserve registry order.

### Error output

Human errors:

```text
error: <message>
hint: <actionable suggestion>
```

JSON errors:

```json
{
  "schemaVersion": 1,
  "command": "source add",
  "kind": "error",
  "data": {
    "error": {
      "code": "manifest_missing",
      "message": "manifest not found in /home/me/not-a-source",
      "hint": "create /home/me/not-a-source/tuck.toml with a valid name"
    }
  },
  "exitCode": 1
}
```

For this slice, implement error mapping for the source/state/resolve kinds that
are already available or needed by source commands:

| Condition | Human/JSON code | Hint |
| --- | --- | --- |
| `manifest.ErrMissing` from `source add <path>` | `manifest_missing` | create `<path>/tuck.toml` with a valid `name` |
| `manifest.ErrInvalid` from `source add <path>` | `manifest_invalid` | fix `<path>/tuck.toml` |
| malformed/unreadable loaded machine state | `state_invalid` | fix or remove the machine-local `sources.toml` |
| `source add <path>` path missing or not a directory before manifest load | `source_root_missing` | pass the path to an existing source repository |
| `resolve.ErrNoSource` from source-resolution plumbing | `no_source` | run `tuck source add <path> --default` or pass `--source <id>` |
| `resolve.ErrUnknownSource` from source-resolution plumbing | `unknown_source` | run `tuck source list` to see enabled sources |
| other runtime I/O write/rename failures | `io_error` | retry after fixing filesystem permissions or disk state |

It is acceptable for M7-M10 acceptance coverage to exercise only the source-add
and source-list rows directly. Keep the resolver mappings covered by unit tests
or narrow application tests so future domain commands can reuse the renderer.

## `source add` behavior

Command:

```text
tuck [--json] source add <path> [--default]
```

Behavior:

1. Require exactly one path argument; let the CLI framework own usage errors.
2. Expand/canonicalize the path as a source root:
   - reject empty path;
   - expand `~`;
   - make absolute;
   - clean lexical components;
   - require an existing directory;
   - evaluate symlinks.
3. Load the source manifest with `manifest.Load(canonicalPath)`.
4. Use the manifest `Name` as the source id. MVP does not support source-id
   overrides.
5. Load the existing registry. An absent registry is an empty registry.
6. Add or update the source entry:
   - if no entry with that id exists, append `{id, path, enabled=true}`;
   - if an entry with that id and the same canonical path exists, update it in
     place and set `enabled=true`;
   - if a disabled entry with that id and the same canonical path exists, re-enable
     it;
   - if an entry with that id exists at a different canonical path, fail before
     writing. Use `state_invalid` unless `cli-spec.md` is updated to add a more
     specific source-id collision code. This preserves the future P12 design for a
     source-id override.
7. If `--default` is passed, set top-level `Registry.Default` to the source id.
8. Validate the complete post-add registry with the same enabled-only rules as
   `state.Load`.
9. Atomically write the normalized state file only after validation succeeds.
10. Print the resulting source registry in the success shape above.

Non-goals:

- no interactive initialization;
- no prompt to resolve id collisions;
- no checksum sidecar;
- no disabled-source management beyond re-enabling a matching id/path entry.

## Machine-state write support

Add state-write support in an engine package rather than embedding TOML write
logic in command actions. One acceptable API is:

```go
package state

func Save(registry Registry) error
func AddSource(path string, makeDefault bool) (Registry, Source, error)
```

Alternative names are fine, but keep these properties:

- write via the state package's sources-file path helper;
- create the parent `tuck` state directory when needed;
- emit normalized TOML with top-level `default` and `[[source]]` entries;
- do not write per-source default flags;
- write `enabled = true/false` explicitly for deterministic output;
- use a temp file in the target directory and `os.Rename` for atomic replacement;
- validate the post-write registry before replacing the existing file;
- wrap state write/validation failures with typed sentinel errors so the app
  layer can map them.

The existing `state.Load` validation logic should not be duplicated loosely.
Prefer extracting a private validator/canonicalizer that both `Load` and write
paths share.

## `source list` behavior

Command:

```text
tuck [--json] source list
tuck [--json] source ls
```

Behavior:

1. Load machine state with `state.Load`.
2. If state is absent or contains no entries, exit `0` and print the empty-state
   message/envelope.
3. Print all recorded entries, including disabled entries.
4. Mark the top-level default by comparing `Registry.Default` to each source id.
5. Do not require an active source.
6. Do not fail just because no source is enabled. Only invalid/unreadable state is
   an error.

## Acceptance coverage

Add a `source` acceptance suite:

- `acceptance/source_test.go` running `testdata/script/source`.
- Update `mise` task wiring only if the current `mise run test:accept` does not
  automatically discover the new suite.

### Red-phase discipline

Prefer a minimal production compile seam for unit-level red tests. Add the small
API surface that the slice needs, with deliberately not-implemented behavior, so
unit tests compile and fail for behavior rather than for missing symbols. Keep
these seams narrow and production-owned; do not add test-only exported APIs just
to satisfy tests.

Recommended initial seams:

```go
package state

func Save(registry Registry) error
func AddSource(path string, makeDefault bool) (Registry, Source, error)
```

The seam implementation can return a typed not-implemented/state error or another
clear failing result at first. The red checkpoint is valid only when the package
compiles and the new tests fail because `Save`/`AddSource` do not yet perform the
specified behavior.

Acceptance scripts should also be added early against the existing CLI surface.
They should invoke `tuck source add`, `tuck source list`, and `tuck source ls`,
which already exist in the command skeleton, so the acceptance test binary
compiles before source command implementation is added.

Recommended red check sequence:

1. Add minimal production compile seams for state write/add behavior.
2. Add unit tests for the seam behavior that should fail against the minimal
   implementation.
3. Add `acceptance/source_test.go`, the `testdata/script/source/*.txtar` files,
   and the `wanthome` top-level-default helper fix.
4. Run:

   ```sh
   go test ./internal/state
   ```

   Confirm the package compiles and the new unit tests fail for expected missing
   behavior, not missing symbols.
5. Run:

   ```sh
   go test -tags tuck_testhooks -run TestSuites/source ./acceptance/...
   ```

6. Confirm the package compiles and the result is a test failure caused by the
   historical command behavior. Acceptable first failures included:
   - `source add` exits with the "not implemented" diagnostic;
   - `source list` exits with the "not implemented" diagnostic;
   - JSON/human stdout/stderr assertions mismatch because the renderer is not
     implemented.
7. If the failure is a compile error, parser error in a txtar script, missing
   helper, wrong environment setup, or stale state fixture shape, fix the red
   tests before starting green implementation.
8. Capture the red output in the implementation notes or PR description so the
   test-first step is auditable.

Suggested red scripts:

1. `list-empty.txtar`
   - no `wanthome`;
   - run `tuck source list --no-color`;
   - assert stdout `no sources enabled`, stderr empty, exit `0`;
   - run `tuck --json source list`;
   - assert one `kind: "sources"` envelope with an empty source list.

2. `add-default-and-list.txtar`
   - create `$WORK/home`;
   - provide `src/tuck.toml` with `name = "public"` and description;
   - run `tuck source add $WORK/src --default --no-color`;
   - assert state file exists and contains top-level `default = "public"`;
   - run `tuck source list --no-color`;
   - assert the source appears with canonical path and default marker.

3. `add-json.txtar`
   - add a source with `--json`;
   - assert exactly one JSON envelope on stdout and empty stderr.

4. `add-missing-manifest.txtar`
   - point at an existing directory without `tuck.toml`;
   - assert human error on stderr, empty stdout, exit `1`;
   - assert JSON variant has `error.code = "manifest_missing"` and empty stderr.

5. `add-invalid-manifest.txtar`
   - provide malformed or missing-name `tuck.toml`;
   - assert `manifest_invalid` and a fix-manifest hint.

6. `add-id-collision.txtar`
   - add `src-a` with `name = "public"`;
   - attempt to add `src-b` with the same `name` and different path;
   - assert failure and verify the original state remains unchanged.

7. `list-invalid-state.txtar`
   - write malformed `state/tuck/sources.toml`;
   - run `source list`;
   - assert `state_invalid` with a repair hint.

Also update the existing `wanthome` helper to generate valid top-level default
state:

```toml
default = "public"

[[source]]
id = "public"
path = "$WORK/src"
enabled = true
```

## Unit coverage

Add or extend unit tests for:

- normalized TOML write shape;
- absent state parent directory creation;
- atomic write success path;
- add-new-source behavior;
- re-enable same id/path behavior;
- reject same id/different path without modifying state;
- write-path validation shares enabled-only validation with `Load`;
- app error mapping for manifest/state/resolve errors;
- `source ls` canonical command name in JSON.

For the initial red checkpoint, prefer adding minimal production compile seams
for the state write/add API and then writing unit tests against those seams. This
keeps the repository buildable while still making the unit tests genuinely red.
Acceptance coverage should still be added in the same red phase because it proves
the user-observable source commands compile and fail before implementation.

## Task breakdown

1. Add minimal state write/add compile seams
   - Add narrow production APIs for `Save` and `AddSource` or equivalent names.
   - Keep the initial implementation intentionally incomplete.
   - Do not add broad abstractions or test-only exported helpers.

2. Add compiling red unit tests
   - Add unit tests for normalized save output, add-new-source behavior,
     re-enable same id/path, id collision, and no-write-on-validation-failure.
   - Run `go test ./internal/state`.
   - Confirm tests compile and fail for behavior, not missing symbols.

3. Add compiling red source acceptance suite
   - Add source-suite coverage to `TestSuites`.
   - Add source scripts for empty list, add/list success, JSON success, manifest
     failures, id collision, and invalid state.
   - Fix `wanthome` to use top-level `default`.
   - Run `go test -tags tuck_testhooks -run TestSuites/source ./acceptance/...`.
   - Confirm the tests compile and fail for expected behavior, not harness or
     syntax errors.

4. Add output/error primitives
   - Define reusable envelope structs for `sources` and `error`.
   - Implement human error rendering to stderr.
   - Implement JSON error rendering to stdout with empty stderr.
   - Map manifest/state/resolve/source-root/write errors to stable codes and
     hints.

5. Add state write/add support
   - Extract shared state validation where needed.
   - Add normalized TOML encoding.
   - Add atomic save.
   - Add source add/update/re-enable/collision behavior.

6. Wire `source add`
   - Parse args and `--default`.
   - Call the state add/write engine.
   - Render human or JSON `sources` output.

7. Wire `source list`
   - Load state.
   - Render empty and non-empty human output.
   - Render JSON `sources` output.
   - Ensure `source ls` emits canonical command metadata.

8. Verify and mark complete
   - Run `go test ./internal/...`.
   - Run `go test -tags tuck_testhooks -run TestSuites/source ./acceptance/...`.
   - Run `mise run test`.
   - Run `go build ./...`, `go vet ./...`, and the gofmt gate.
   - Mark M7, M8, M9, and M10 complete in `docs/backlog.md` only after green
     verification.

## Acceptance criteria

- `source add` writes machine state atomically and does not write on validation
  failure.
- `source add --default` writes top-level `default = "<id>"`.
- Adding the same id/path is idempotent and re-enables disabled entries.
- Adding the same id with a different canonical path fails without changing state.
- `source list` succeeds for absent/empty state.
- `source list` shows all recorded entries and marks the registry default.
- Human errors go to stderr with an actionable `hint:`.
- JSON mode emits exactly one envelope on stdout and keeps stderr empty.
- Source command JSON uses `kind: "sources"` and canonical command names.
- Error JSON uses `kind: "error"` with stable `data.error.code`.
- Existing foundation acceptance tests still pass.
