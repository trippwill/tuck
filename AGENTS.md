# Repository instructions for agents

## Commands

- Build production packages without test hooks: `go build ./...`
- Run all tests via mise: `mise run test`
- Run unit tests only: `mise run test:unit` or `go test ./internal/...`
- Run acceptance tests only: `mise run test:accept` or `go test -tags tuck_testhooks ./acceptance/...`
- Run the current acceptance suite: `mise run test:accept:foundation`
- Run a single unit test: `go test -run TestRun ./internal/cli`
- Run one acceptance suite directly: `go test -tags tuck_testhooks -run TestFoundation ./acceptance/...`
- Vet/format gates described in docs: `go vet ./...` and `test -z "$(gofmt -l ./cmd ./internal ./acceptance)"`

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
- Test-only seams live behind the `tuck_testhooks` build tag in `internal/testhooks`. Production builds must work tagless and must not depend on `TUCK_TEST_*` behavior.

## Repository-specific conventions

- The CLI is spec-first. The important user model is one active source at a time: a committed `<repo>/tuck.toml` manifest supplies the source id, while machine-local `${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml` records enabled paths/defaults.
- Command depth mirrors operation frequency: file operations (`adopt`, `eject`, `status`) are top-level; package operations (`package use`, `package drop`, etc.) are grouped under the `package` (alias `pkg`) subcommand; source operations are grouped under the `source` subcommand.
- No false globals: flags appear only on commands where they have meaning. `--json` and `--no-color` are truly universal (on the root command). `--source` and `--root` are scoped to domain commands (file ops + package subcommands). `--apply` is scoped to mutating commands only.
- Exit codes are binary: `0` (success) or `1` (failure). Error classification lives in the `--json` error envelope (`error.code`) and stderr messages, not in distinct exit codes.
- Package refs are plain package names only. Do not encode source or context in refs; source comes from `--source`/machine state and context comes from `--root` or default `home`.
- Repo `tuck.toml` owns portable package/file policy. Use keyed package metadata (`[package.<name>]` with `[[package.<name>.file]]`) for per-file deploy strategy and explicit modes; do not prefer package-local metadata files.
- Mutating verbs (`adopt`, `eject`, `package use`, `package drop`, `package refresh`) are dry-run/plan-by-default and mutate only with `--apply`. Build the complete plan and accumulate all conflicts before any mutation.
- Output contract matters for domain commands: primary results and JSON envelopes go to stdout; diagnostics and hints go to stderr. Framework-rendered help/usage follows urfave/cli defaults. With `--json`, emit exactly one envelope on stdout and keep stderr empty.
- Prefer `internal/apperr` for stable package-level error kinds. Define a small string sentinel type that implements `Error()`, expose const values, alias the package error (`type Error = apperr.Error[ErrKind]`), and wrap causes with `apperr.Wrap`, `Wrapf`, or `WrapErr` so `errors.Is` and `errors.As` work.
- Path handling must be path-segment aware. Ownership is inferred from symlink payloads within the active source only; there is no deployed-link manifest.
- `package use` links only leaf entries. Directory entries become real target directories, and symlink payloads should use the spec's relative form.
- Copied-file deployment (`deploy = "copy"`) is state-backed because ownership cannot be inferred from symlink payloads. Track copied entries and checksums in machine-local state; report drift rather than overwriting/removing changed files silently.
- Machine-local state remains human-readable text. Use a generated checksum sidecar/field for fast validation; do not replace state with opaque binary storage.
- Root-context behavior separates logical paths, physical test backing roots, and privilege authorization. `--root` output should stay logical (`/etc/...`), while tests may redirect writes with build-tagged hooks.
- Acceptance scripts should run in the harness sandbox, use `exec` for success and `! exec` for expected failures, assert symlink payloads via `readlink`, and generate `$WORK`-dependent state at runtime with setup commands such as `wanthome` rather than inline txtar bodies.
- Keep acceptance output deterministic: scrubbed sandbox env, stable locale, no color, fixed umask from the harness, stdout/stderr assertions, and no dependence on the developer's real home, root, or machine state.
- `codebook.toml` is the project-local spell-check dictionary; add legitimate project terms there instead of working around spelling checks.
