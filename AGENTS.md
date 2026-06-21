# Repository instructions for agents

## Commands

- Build production packages without test hooks: `go build ./...`
- Generate checked-in generated files: `mise run generate`
- Build production packages through mise, after generation and vet: `mise run build`
- Run all tests via mise: `mise run test`
- Run unit tests only: `mise run test:unit` or `go test ./internal/...`
- Run command-package tests only: `mise run test:cmd` or `go test ./cmd/...`
- Run acceptance tests only: `mise run test:accept` or `go test -tags tuck_testhooks ./acceptance/...`
- Run one acceptance suite: `mise run test:accept:package` or `go test -tags tuck_testhooks -run TestSuites/package ./acceptance/...`
- Run a single unit test: `go test -run TestRunMetaJSONUsesSharedEnvelope ./internal/app`
- Run one acceptance suite directly: `go test -tags tuck_testhooks -run TestSuites/source ./acceptance/...`
- Run the full local gate: `mise run check`
- Print unit coverage without enforcing a threshold: `mise run coverage`
- Vet/format gates individually: `mise run vet` and `mise run fmt`

## Commit messages

- Use Conventional Commits: `<type>(<scope>): <summary>`, for example `feat(cli): add source add`.
- Prefer scopes that match the touched area, such as `cli`, `manifest`, `state`, `pathutil`, `acceptance`, `docs`, or `build`.
- Use `test:` for test-only changes and `docs:` for documentation-only changes.

## Architecture

- `cmd/tuck/main.go` is intentionally thin: it delegates to the app-level urfave/cli entrypoint. The acceptance harness registers the same entrypoint with `testscript` so command stdout, stderr, and exit status are tested as user-observable behavior.
- `internal/app` owns the current CLI skeleton on `urfave/cli/v3`: global flags, subcommands, help metadata, framework-owned shell errors, and command exit handling. Prefer urfave/cli defaults and `cli.ExitCoder` patterns over bespoke CLI plumbing unless tuck-specific semantics require otherwise.
- `docs/cli-spec.md` is the authoritative product/algorithm spec. It defines command semantics, source selection, path resolution, planning, output envelopes, and exit codes. Match the spec unless there is an intentional doc update in the same change.
- `docs/testing-strategy.md` is authoritative for how behavior is tested. The backlog expects red/green, vertical-slice delivery: add the failing unit or acceptance coverage for a behavior before implementing it.
- Engine code is expected to grow under focused `internal/...` packages. The docs call out units such as manifest/state loading, path primitives, source/package resolution, ownership inference, conflict rules, and planning; keep pure algorithmic logic out of the CLI layer where practical.
- Acceptance tests live in `acceptance/` and use `github.com/rogpeppe/go-internal/testscript`. `TestMain` registers a `tuck` command backed by the already-compiled test binary, so scripts should invoke `tuck` directly rather than `go run`, `go build`, or an external binary.
- Test-only seams live behind the `tuck_testhooks` build tag in the package that owns the behavior. Production builds must work tagless and must not depend on `TUCK_TEST_*` behavior.

## Repository-specific conventions

- The CLI is spec-first. The important user model is one active source at a time: a committed `<repo>/.tuck.toml` manifest supplies the source id, while machine-local `${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml` records enabled paths/defaults.
- Command depth mirrors operation frequency: file operations (`adopt`, `eject`, `status`) are top-level; package operations (`package use`, `package drop`, etc.) are grouped under the `package` (alias `pkg`) subcommand; source operations are grouped under the `source` subcommand.
- `--json` and `--no-color` are universal. `--source` and `--root` are root-level selectors for before-command placement, but only domain commands (file ops + package subcommands) use them; source commands warn when they are ignored. `--apply` is scoped to mutating commands only.
- Exit codes are binary: `0` (success) or `1` (failure). Error classification lives in the `--json` error envelope (`error.code`) and stderr messages, not in distinct exit codes.
- Package refs are plain package names only. Do not encode source or context in refs; source comes from `--source`/machine state and context comes from `--root` or default `home`.
- Package-local `.tuck.toml` owns portable package/file policy. Use `[[file]]` metadata inside `<package-root>/.tuck.toml` for per-file deploy strategy and explicit modes; source-level `.tuck.toml` owns source identity only.
- Mutating verbs (`adopt`, `eject`, `package use`, `package drop`, `package refresh`) are dry-run/plan-by-default and mutate only with `--apply`. Build the complete plan and accumulate all conflicts before any mutation.
- Output contract matters for domain commands: primary results and JSON envelopes go to stdout; diagnostics and hints go to stderr. Framework-rendered help/version follows urfave/cli defaults even with `--json`. Domain commands with `--json` emit exactly one envelope on stdout and keep stderr empty.
- Prefer typed const sentinel errors for stable package-level error kinds. Define one small string sentinel type per package, add a one-line `Error()` method, and call `internal/apperr` generic helpers directly. Use `AppErrMsg` for context-only errors and `AppErrWrap` when preserving a cause so `errors.Is` and `errors.As` work.
- Path handling must be path-segment aware. Symlink ownership is inferred from payloads within the active source only; copied-file ownership is state-backed.
- `package use` deploys only leaf entries. Directory entries become real target directories, and symlink payloads should use the spec's relative form.
- Copied-file deployment (`deploy = "copy"`) is state-backed because ownership cannot be inferred from symlink payloads. Track copied entries, checksums, and applied modes in machine-local state. `package use` is conservative on drift; `package refresh` remains strict drop-then-use, so source-only drift can refresh but target drift conflicts until the user removes the target or adopts it.
- Machine-local state remains human-readable text. Use a generated checksum sidecar/field for fast validation; do not replace state with opaque binary storage.
- Root-context behavior separates logical paths, physical test backing roots, and privilege authorization. `--root` output should stay logical (`/etc/...`), while tests may redirect writes with build-tagged hooks.
- Acceptance scripts should run in the harness sandbox, use `exec` for success and `! exec` for expected failures, assert symlink payloads via `readlink`, and generate `$WORK`-dependent state at runtime with setup commands such as `wanthome` rather than inline txtar bodies.
- Keep acceptance output deterministic: scrubbed sandbox env, stable locale, no color, fixed umask from the harness, stdout/stderr assertions, and no dependence on the developer's real home, root, or machine state.
- `.codebook.toml` is the project-local spell-check dictionary; add legitimate project terms there instead of working around spelling checks.
