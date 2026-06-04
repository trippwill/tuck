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

- Use Conventional Commits: `<type>(<scope>): <summary>`, for example `feat(cli): add source enable`.
- Prefer scopes that match the touched area, such as `cli`, `manifest`, `state`, `pathutil`, `acceptance`, `docs`, or `build`.
- Use `test:` for test-only changes and `docs:` for documentation-only changes.

## Architecture

- `cmd/tuck/main.go` is intentionally thin: it calls `internal/cli.Run(os.Args, os.Environ(), stdout, stderr)` and exits with the returned code. Keep process exit behavior out of internal packages so tests can call `Run` directly.
- `internal/cli` owns the current CLI skeleton on `urfave/cli/v3`: global flags, subcommands, custom help, usage errors, and exit-code handling. `Run` writes primary output to the injected stdout writer and diagnostics to stderr.
- `docs/cli-spec.md` is the authoritative product/algorithm spec. It defines command semantics, source selection, path resolution, planning, output envelopes, and exit codes. Match the spec unless there is an intentional doc update in the same change.
- `docs/testing-strategy.md` is authoritative for how behavior is tested. The backlog expects red/green, vertical-slice delivery: add the failing unit or acceptance coverage for a behavior before implementing it.
- Engine code is expected to grow under focused `internal/...` packages. The docs call out units such as manifest/state loading, path primitives, source/package resolution, ownership inference, conflict rules, and planning; keep pure algorithmic logic out of the CLI layer where practical.
- Acceptance tests live in `acceptance/` and use `github.com/rogpeppe/go-internal/testscript`. `TestMain` registers a `tuck` command backed by the already-compiled test binary, so scripts should invoke `tuck` directly rather than `go run`, `go build`, or an external binary.
- Test-only seams live behind the `tuck_testhooks` build tag in `internal/testhooks`. Production builds must work tagless and must not depend on `TUCK_TEST_*` behavior.

## Repository-specific conventions

- The CLI is spec-first. The important user model is one active source at a time: a committed `<repo>/tuck.toml` manifest supplies the source id, while machine-local `${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml` records enabled paths/defaults.
- Package refs are plain package names only. Do not encode source or context in refs; source comes from `--source`/machine state and context comes from `--root` or default `home`.
- Mutating verbs (`deploy`, `undeploy`, `redeploy`, `adopt`, `eject`) are dry-run/plan-by-default and mutate only with `--apply`. Build the complete plan and accumulate all conflicts before any mutation.
- Output contract matters: primary results and JSON envelopes go to stdout; diagnostics, hints, and usage errors go to stderr. With `--json`, emit exactly one envelope on stdout and keep stderr empty.
- Preserve the exit-code taxonomy from `internal/cli/exit.go` and `docs/cli-spec.md`: `0` OK, `1` conflict, `2` usage, `3` config/state, `4` resolution, `5` privilege, `6` runtime.
- Path handling must be path-segment aware. Ownership is inferred from symlink payloads within the active source only; there is no deployed-link manifest.
- Deploy links only leaf entries. Directory entries become real target directories, and symlink payloads should use the spec's relative form.
- Root-context behavior separates logical paths, physical test backing roots, and privilege authorization. `--root` output should stay logical (`/etc/...`), while tests may redirect writes with build-tagged hooks.
- Acceptance scripts should run in the harness sandbox, use exact exit assertions via `wantexit`, assert symlink payloads via `readlink`, and generate `$WORK`-dependent state at runtime with setup commands such as `wanthome` rather than inline txtar bodies.
- Keep acceptance output deterministic: scrubbed sandbox env, stable locale, no color, fixed umask from the harness, stdout/stderr assertions, and no dependence on the developer's real home, root, or machine state.
