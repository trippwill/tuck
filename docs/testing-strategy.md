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
  `$HOME`, source directories, a generated config). Tests never read or write
  the developer's real home, real `/`, or real config.
- **Deterministic.** Same inputs ⇒ identical bytes out. No dependence on the
  host environment, locale, umask, clock, or ordering.

## 2. Test layers

| Layer | Scope | Speed | Tooling |
| --- | --- | --- | --- |
| **Unit** | Pure functions from [§12](./cli-spec.md#12-resolution-algorithms): path primitives, target↔package conversion, classify-target, ownership inference, conflict rules, execution planning. No filesystem, or `t.TempDir()` only. | fast | stdlib `testing`, table-driven |
| **Acceptance** | The compiled binary against an isolated filetree. Asserts exit code ([§10](./cli-spec.md#10-exit-codes)), stdout (human or JSON, [§9](./cli-spec.md#9-output-formats)), and the resulting filetree (existence, symlink payload, moved files). | medium | `testscript` (txtar) |

Unit tests own the algorithmic edge cases; acceptance tests own the
user-observable contract. A behavior is "done" only when an acceptance test
proves it on the real binary.

## 3. Acceptance harness (`testscript`)

We use [`github.com/rogpeppe/go-internal/testscript`](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript).
It is purpose-built for "compiled binary + isolated `$WORK` filetree + golden
stdout":

- `TestMain` registers `tuck`'s `main` so each script invokes the **real
  program** (the test binary dispatches to `main`); scripts never pick up a
  `tuck` from `$PATH`.
- Each script (`testdata/script/*.txtar`) gets a fresh `$WORK` temp directory.
- A txtar archive carries the input tree (source packages, config) and golden
  files inline.
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
- `wantexit <N> <tuck-args...>` — run `tuck` and assert the **exact** process
  exit code is `N`. The builtin `! tuck …` only asserts *non-zero*, which is
  insufficient for distinguishing `1`/`4`/`5`/`6`.
- `wanthome` / `wantroot` — convenience setup that scaffolds the fake `$HOME`
  (or root backing tree) and a config file.

> JSON tests can alternatively assert the envelope's `exitCode` field
> ([§9.2](./cli-spec.md#92-json-output)), which mirrors the process code, by
> comparing against a golden document.

## 4. The isolated filetree

Each script builds a sandbox under `$WORK`:

```
$WORK/
  home/                 # fake $HOME (home-context target root)
  root/                 # fake physical backing root (root-context only; see §5)
  src/                  # a source directory
    zsh/.zshrc          # a home-context package
    .root/sshd/etc/...  # a root-context package (base is <source>/.root)
  config.toml           # generated; [sources.public] path = $WORK/src
```

Every script **creates `$HOME` itself** (the tool must not be relied on to
create the target root) and sets a scrubbed environment:

| Variable | Value | Why |
| --- | --- | --- |
| `HOME` | `$WORK/home` | home-context target root |
| `TUCK_CONFIG` | `$WORK/config.toml` | deterministic config discovery |
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
   **mutate nothing**, exit `5`.
5. Otherwise apply. A genuine filesystem error *during* apply is exit `6`, never
   `5` — the privilege failure path must mutate nothing.

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
- Exit `5` ⟺ `--apply && required && !satisfied`.

> This matches [§8.1](./cli-spec.md#81-privilege-root-context) (privilege as an
> explicit preflight policy) and the `privilege` object in
> [§9.2](./cli-spec.md#92-json-output).

## 6. What every acceptance test asserts

1. **Exit code** — exact, via `wantexit` or the JSON `exitCode` field.
2. **Stdout** — golden human text (`--no-color`) or a golden JSON document.
3. **Filetree** — `exists` / `! exists`, and `readlink` for the **exact**
   symlink payload. The spec's intended payload form (relative vs absolute) is
   asserted explicitly; output is never used to infer the payload.

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
   invocation, golden output, exit code, and filetree. It fails (the feature
   isn't built, or `wantexit` mismatches).
2. **Green.** Implement until the script passes.
3. Refactor with the script as the safety net.

This pairs naturally with **plan-by-default**: a single script first runs the
command **without `--apply`** and asserts the filetree is **unchanged** (the
plan is printed, nothing mutates), then runs **with `--apply`** and asserts the
mutation. The no-mutation guarantee is itself a first-class assertion.

### Coverage map

- One acceptance script per command in [§7](./cli-spec.md#7-command-reference)
  (`deploy`/`undeploy`/`redeploy`/`adopt`/`eject`/`packages`/`tree`/`status`),
  in both plan and `--apply` form for mutating verbs.
- One script per non-zero exit code in [§10](./cli-spec.md#10-exit-codes):
  conflict (`1`), usage (`2`), config (`3`), resolution (`4`), privilege (`5`),
  runtime (`6`).
- One script per conflict rule in
  [§12.6](./cli-spec.md#126-conflict-rules).
- JSON variants for at least one representative of each `kind`
  (`plan`/`packages`/`tree`/`status`/`error`).

## 8. Example (home deploy, red→green)

```
# deploy_home.txtar
env HOME=$WORK/home
env TUCK_CONFIG=$WORK/config.toml
mkdir $WORK/home

# plan only: nothing changes
tuck deploy zsh --no-color
! exists $WORK/home/.zshrc

# apply: link is created with the expected payload
wantexit 0 tuck deploy zsh --apply --no-color
exists $WORK/home/.zshrc
readlink $WORK/home/.zshrc $WORK/src/zsh/.zshrc

-- config.toml --
[sources.public]
path = "$WORK/src"
enabled = true

-- src/zsh/.zshrc --
# zshrc contents
```

## 9. CI

- Build the binary once; run unit tests, then acceptance tests (with
  `-tags tuck_testhooks`).
- Gate `vet` and `gofmt`/lint.
- The OS/arch matrix (linux+macos, amd64+arm64) is a First-Release concern; MVP
  CI runs the developer platform only.
