# tuck — Testing Strategy

Status: design (no implementation yet). Companion to
[`cli-spec.md`](./cli-spec.md), which is authoritative for behavior; this
document is authoritative for **how that behavior is tested**.

## 1. Goals

- **Red/green TDD.** Write the test that describes a behavior **first**; watch it
  fail (red) because the feature is unimplemented; implement until it passes
  (green). Every backlog story lands with its test written first.
- **Test the compiled binary.** Acceptance tests drive the real `tuck` program
  end to end — exit code, stdout, and the resulting filesystem — not internal
  Go APIs.
- **Isolated filetree.** Each test runs against a throwaway temp tree (a fake
  `$HOME`, a source repo, generated machine state). Tests never read or write
  the developer's real home, real `/`, or real machine state.
- **Deterministic.** Same inputs ⇒ identical bytes out. No dependence on the
  host environment, locale, umask, clock, or ordering.

## 2. Test layers

| Layer | Scope | Speed | Tooling |
| --- | --- | --- | --- |
| **Unit** | Pure/internal behavior from [§12](./cli-spec.md#12-resolution-algorithms): path primitives, target↔package conversion, classify-target, ownership inference, conflict rules, execution planning. No filesystem, or `t.TempDir()` only. | fast | stdlib `testing`, table-driven |
| **Acceptance** | The compiled binary against an isolated filetree. Asserts exit code ([§10](./cli-spec.md#10-exit-codes-and-error-codes)), stdout/stderr (human or JSON, [§9](./cli-spec.md#9-output-formats)), and the resulting filetree (existence, symlink payload, moved files). | medium | `testscript` (txtar) |

Unit tests own the algorithmic edge cases; acceptance tests own the
user-observable contract. A behavior is "done" only when an acceptance test
proves it on the real binary.

CLI shell behavior — command parsing, help/version output, diagnostics, stdout,
stderr, and process exit status — is user-observable contract and belongs in
testscript, not in unit tests for the CLI composition package.

### 2.1 Test suites

Tests are grouped into **named suites** that each run independently, so you can
exercise one slice without running the whole tree. A suite is a single Go test
function, which keeps `go test -run` filtering and IDE "run test" gutters
working:

- **Unit suites** are one Go package per engine concern (e.g.
  `internal/manifest`, `internal/state`, `internal/resolve`, `internal/plan`,
  `internal/pathutil`). Run one with `go test ./internal/plan/...`.
- **Acceptance suites** are one subdirectory per slice under
  `testdata/script/<suite>/`, each driven by its own `Test<Suite>` function that
  points `testscript` at that directory. Run one with
  `go test -tags tuck_testhooks -run TestDeploy ./acceptance/...`. Current
  suites track the thin vertical slices ([backlog.md](./backlog.md) MVP):
  `source` (`add`/`list`, state validation, resolution, `no_source`),
  `package_use`, `query` (`package list`/`package show`/`status`),
  `package_drop` (incl. `package refresh`),
  `adopt_eject`, `root` (root context + privilege; needs the `TUCK_TEST_ROOT_DIR`
  / `TUCK_TEST_PRIVILEGE` hooks), and `errors` (error-code classification and JSON
  `error`/`sources`/… envelopes, cross-cutting).

[`mise`](https://mise.jdx.dev) tasks expose the common groupings: `mise run test`
(everything), `mise run test:unit`, `mise run test:accept` (all acceptance, with
the tag), and `mise run test:accept:<suite>` (one suite). CI runs `mise run test`
plus a tagless `go build ./...` smoke (see §9). `mise` also pins the Go toolchain
version for reproducible local and CI builds.

## 3. Acceptance harness (`testscript`)

We use [`github.com/rogpeppe/go-internal/testscript`](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript).
It is purpose-built for "compiled binary + isolated `$WORK` filetree + golden
stdout":

- `TestMain` registers `tuck`'s entrypoint so each script invokes the **real
  program** within the **same already-compiled test binary** (testscript's
  `RunMain` dispatches the registered command — no `$PATH` lookup, no separately
  built binary). Because that binary is compiled with `-tags tuck_testhooks`, the
  test hooks are active for the `tuck` command; scripts must therefore never
  shell out to `go run`/`go build`/an external `tuck`, which would not carry the
  tag.
- Each script (`testdata/script/<suite>/*.txtar`) gets a fresh `$WORK` temp
  directory.
- A txtar archive carries the **static** input tree inline (the source repo and
  its `tuck.toml`, package contents). Files that must embed an absolute sandbox
  path (machine state `sources.toml`) are generated at run time by a setup
  command (§3.2), since testscript does not substitute `$WORK` in archive bodies.
- `testscript` automatically scrubs `$WORK` from output before golden
  comparison, so absolute temp paths don't leak into goldens.

### 3.1 Build-tag–gated test hooks

The harness needs to redirect two things the spec ties to the host: the root
context's physical target root (`/`) and the privilege decision. These are
exposed **only** through code compiled under the `tuck_testhooks` build tag:

```go
//go:build tuck_testhooks
```

- The acceptance harness builds/runs the test binary **with** `-tags
  tuck_testhooks`.
- Release builds (`go build` with no tag) **physically omit** this code, so a
  production binary cannot honor any `TUCK_TEST_*` variable, even if one leaks
  into its environment (e.g. under `sudo` or in CI). This removes the footgun of
  a silent production override.

### 3.2 Custom script commands

`testscript` builtins cover most needs (`exec`, `exists`, `! exists`, `cmp`,
`stdin`, `env`, `mkdir`, `chmod`). We register a few custom commands:

- `readlink <path> <expected-payload>` — assert a symlink exists and its raw
  payload equals `<expected-payload>` exactly (relative vs absolute matters —
  see §6).
- `wanthome` / `wantroot` — convenience setup that scaffolds the fake `$HOME`
  (or root backing tree) and **generates the `$WORK`-dependent machine state**
  (`state/tuck/sources.toml`, with the default source pointing at `$WORK/src`)
  at run time. Files whose contents must embed an absolute sandbox path **cannot**
  be carried as inline txtar bodies — testscript only substitutes `$WORK` in
  command lines, not in archive file contents — so they are written by these
  setup commands. Static files (`tuck.toml`, package contents) stay inline.

> JSON tests can alternatively assert the envelope's `exitCode` field
> ([§9.2](./cli-spec.md#92-json-output)), which mirrors the process code, by
> comparing against a golden document.

## 4. The isolated filetree

Each script builds a sandbox under `$WORK`:

```
$WORK/
  home/                 # fake $HOME (home-context target root)
  root/                 # fake physical backing root (root-context only; see §5)
  src/                  # a source repo
    tuck.toml           # committed manifest; name = "public"
    zsh/.zshrc          # a home-context package
    .root/sshd/etc/...  # a root-context package (base is <source>/.root)
  state/                # machine-local state dir (TUCK_TEST_STATE_DIR)
    tuck/sources.toml   # GENERATED at run time by wanthome/wantroot, not inline
                        #   (its [[source]] path must be the absolute $WORK/src)
```

> Inline txtar bodies are written verbatim; testscript substitutes `$WORK` only
> in command lines. So any file that must reference an absolute sandbox path
> (notably `sources.toml`) is generated by a setup command, while static files
> (`tuck.toml`, package contents) are carried inline.

Every script **creates `$HOME` itself** (the tool must not be relied on to
create the target root) and sets a scrubbed environment:

| Variable | Value | Why |
| --- | --- | --- |
| `HOME` | `$WORK/home` | home-context target root |
| `TUCK_TEST_STATE_DIR` | `$WORK/state` | deterministic machine-state discovery (build tag `tuck_testhooks`) |
| `XDG_STATE_HOME` | `$WORK/state` | prevent fallback to a real XDG state path |
| `XDG_CONFIG_HOME` | `$WORK/xdg` | prevent fallback to a real XDG path |
| `NO_COLOR` | `1` | stable output (tests also pass `--no-color`) |
| `TERM` | `dumb` | no terminal escape sequences |
| `LANG` / `LC_ALL` | `C` | stable sorting and messages |
| `TUCK_TEST_ROOT_DIR` | unset (home tests) / `$WORK/root` (root tests) | §5 |
| `TUCK_TEST_PRIVILEGE` | unset / `granted` / `denied` | §5 |

A fixed `umask` is set in the harness so any directory-mode assertions are
reproducible.

## 5. Root context: three separate concerns

The root context is the only hard isolation problem, because
`targetRoot(root) = /`. The rubber-duck review showed that conflating "where
root writes go" with "is this allowed" is wrong, so we keep **three independent
concerns** separate. Only the first two have test hooks; the third is a normal
product policy with an injectable check.

### 5.1 Logical context (unchanged)

`--root` still means the `root` context with package base `<source>/.root`. All
**CLI-visible paths stay logical**: plan/JSON/human output for a root-context
operation prints `/etc/ssh/sshd_config`, never the physical sandbox path. This
guarantees acceptance goldens match production output.

### 5.2 Physical backing root (test hook: `TUCK_TEST_ROOT_DIR`)

Under the `tuck_testhooks` tag, the physical root for filesystem operations is:

- production: `/`;
- tests: `$TUCK_TEST_ROOT_DIR` (e.g. `$WORK/root`).

So a logical target `/etc/ssh/sshd_config` is physically created at
`$TUCK_TEST_ROOT_DIR/etc/ssh/sshd_config`. To make logical-vs-physical bugs
visible, JSON output includes the **effective physical root** as a debug field
**only when the hook is active**; production JSON omits it. Root-context tests
assert the logical path in output **and** use `readlink` on the physical path —
never inferring the symlink payload from output alone.

### 5.3 Privilege authorization (policy, not writability)

Privilege is a **preflight policy**, decided before any mutation, and is **not**
derived from whether the target root happens to be writable (writability of `/`
is neither necessary nor sufficient — a plan may touch read-only subtrees, and
`remove_symlink`/`move` depend on parent directories, not the root):

1. Build the conflict-free plan.
2. If root context and the plan contains write actions
   (`mkdir`/`symlink`/`remove_symlink`/`move`), the plan is marked as requiring
   privilege.
3. The privilege check is a single injectable predicate. Production: the process
   is privileged (e.g. `euid == 0`, or holds the needed capability). Tests
   (under the tag): `TUCK_TEST_PRIVILEGE=granted|denied` forces the answer
   deterministically — **no reliance on `chmod 0555`** (which root can bypass)
   or on the test runner's real euid.
4. If `--apply` is given and privilege is **not** satisfied: print the plan,
   **mutate nothing**, exit `1` with error.code `privilege_required`.
5. Otherwise apply. A genuine filesystem error *during* apply is error.code
   `io_error`, never `privilege_required` — the privilege failure path must
   mutate nothing.

This decoupling means redirecting the physical root (§5.2) never grants
privilege, and forcing privilege never changes where bytes land.

### 5.4 Privilege in output

To avoid the contradiction of "privileges required" alongside a successful
non-root apply, output distinguishes *marker* from *enforcement*:

```json
"privilege": { "required": true, "satisfied": true, "reason": "root-context write" }
```

- `required` — context is `root` and the plan has write actions (informational).
- `satisfied` — the privilege predicate's result.
- Failure with `error.code = "privilege_required"` ⟺ `--apply && required && !satisfied`.

> This matches [§8.1](./cli-spec.md#81-privilege-root-context) (privilege as an
> explicit preflight policy) and the `privilege` object in
> [§9.2](./cli-spec.md#92-json-output).

## 6. What every acceptance test asserts

1. **Exit status** — `exec tuck ...` for success and `! exec tuck ...` for
   expected failure. Since process exits are binary (`0`/`1`), tests assert
   detailed failure classification via stderr or the JSON `error.code` /
   `exitCode` fields rather than a custom exit-code command.
2. **Stdout** — golden human text (`--no-color`) or a golden JSON document.
   Primary results only (plans, listings, status, the JSON envelope).
3. **Stderr** — diagnostics (`error:`/`hint:` lines, usage text) land on
   **stderr**, not stdout ([§9](./cli-spec.md#9-output-formats)). Error scripts
   assert a stderr golden; success scripts assert stderr is **empty**. `--help`
   and usage text are checked **loosely** (exit code + a key substring), never
   pinned verbatim, so a CLI-framework (`urfave/cli`) upgrade does not churn
   goldens. CLI shell cases such as `--help`, `--version`, unknown commands, and
   unknown flags are covered here rather than through internal package unit
   tests.
4. **Filetree** — `exists` / `! exists`, and `readlink` for the **exact**
   symlink payload (the spec's **relative** form, e.g. `../src/zsh/.zshrc`);
   output is never used to infer the payload.

### Determinism checklist

- Set the full env table (§4); create `$HOME` before invoking `tuck`.
- Clear `TUCK_TEST_*` in home-context scripts.
- Set a known `umask` if asserting directory modes.
- JSON goldens rely on struct field order (stable); never on Go map order.
- Include at least one script where the invocation `cwd` is **outside** `$HOME`
  and the source, to prove cwd-independence (except for explicitly relative
  inputs).
- Guard/skip any real-permission test when `euid == 0`.

## 7. Red/green workflow

For each behavior:

1. **Red.** Add `testdata/script/<feature>.txtar` describing the desired
   invocation, golden output, exit status, and filetree. It fails because the
   feature is not built yet or the output/filetree/status assertions mismatch.
2. **Green.** Implement until the script passes.
3. Refactor with the script as the safety net.

This pairs naturally with **plan-by-default**: a single script first runs the
command **without `--apply`** and asserts the filetree is **unchanged** (the
plan is printed, nothing mutates), then runs **with `--apply`** and asserts the
mutation. The no-mutation guarantee is itself a first-class assertion.

### Coverage map

Coverage is organized by **suite** (§2.1); each suite owns the slice of the
contract below.

- **`source`** — `source add`/`source list`; state validation (unique enabled
  ids, ≤1 default, no overlapping roots); active-source resolution and
  `no_source`; `sources` JSON kind.
- **`package_use`** — `package use` in both plan and `--apply` form; the
  no-mutation guarantee; package-use/dir conflicts.
- **`query`** — `package list`/`package show`/`status`/`package status`.
- **`package_drop`** — `package drop` and `package refresh` (payload normalization).
- **`metadata`** — repo `tuck.toml` package/file metadata parsing:
  `[package.<name>]`, `[[package.<name>.file]]`, `deploy`, and explicit `mode`;
  unknown metadata remains additive/forward-compatible.
- **`copy`** — `deploy = "copy"` entries in plan and `--apply` form; copied
  targets are regular files, explicit modes are applied, unsafe overwrites are
  conflicts, and copied entries are recorded in machine-local state.
- **`copy_drift`** — copied-file ownership and drift reporting in `status` and
  `package status`: unchanged, source modified, target modified, both modified,
  and untracked target conflicts.
- **`state_integrity`** — text state plus checksum sidecar validation: valid
  state passes, modified/truncated state reports `state_checksum_mismatch`, and
  the error includes repair guidance.
- **`adopt_eject`** — `adopt` and `eject`; active-source ownership inference.
- **`root`** — `root` context via the physical-root seam with logical-path
  goldens; deterministic privilege via the injected predicate, covering
  `privilege_required` vs `io_error`.
- **`errors`** (cross-cutting) — scripts for stable error.code values in
  [§10](./cli-spec.md#10-exit-codes-and-error-codes); separate foundation scripts
  cover CLI parse/dispatch errors such as unknown commands and flags. Include one
  script per conflict rule in [§12.6](./cli-spec.md#126-conflict-rules);
  JSON variants for at least one representative of each `kind`
  (`plan`/`packages`/`tree`/`status`/`sources`/`help`/`version`/`error`).

Mode assertions should use an acceptance helper or direct testscript stat
assertion that is stable under the harness umask. Copied-file tests must assert
both filesystem bytes and recorded state so that copy ownership is not inferred
accidentally from path shape.

## 8. Example (home package use, red→green)

```
# package_use_home.txtar
wanthome                  # creates $WORK/home and generates $WORK/state/tuck/
                          #   sources.toml with the default source -> $WORK/src

# plan only: nothing changes
tuck pkg use zsh --no-color
! exists $WORK/home/.zshrc

# apply: link is created with the expected (relative) payload
exec tuck pkg use zsh --apply --no-color
exists $WORK/home/.zshrc
readlink $WORK/home/.zshrc ../src/zsh/.zshrc

-- src/tuck.toml --
name = "public"

-- src/zsh/.zshrc --
# zshrc contents
```

The payload is `../src/zsh/.zshrc`, i.e.
`relativePath(dirname($WORK/home/.zshrc), $WORK/src/zsh/.zshrc)`, matching the
spec's relative-payload rule
([§12.7](./cli-spec.md#127-operation-algorithms)) — never an absolute path.

## 9. CI

- Run `mise run test`: unit suites, then acceptance suites (with
  `-tags tuck_testhooks`).
- Add a **tagless** `go build ./...` (and `go vet ./...`) so an accidental
  production dependency on `tuck_testhooks`-only code fails the build — the test
  hooks must never be reachable without the tag.
- Gate `vet` and `gofmt`/lint.
- Acceptance suites run non-parallel (a fixed process `umask` is set once in the
  acceptance package's `TestMain`; per-script `umask` changes would otherwise
  race).
- The OS/arch matrix (linux+macos, amd64+arm64) is a First-Release concern; MVP
  CI runs the developer platform only.
