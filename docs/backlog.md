# tuck — User Story Backlog

A dependency-ordered backlog. Work top-to-bottom: the first unstarted story is
always the next one you can build. Stories are grouped into three milestones;
horizontal rules mark the **MVP cutoff** and the **First Release cutoff**.
Derived from [`cli-spec.md`](./cli-spec.md).

**Ordering.** The MVP opens with a **red/green foundation** — the harness and a
failing first contract suite (M1) before the CLI skeleton that turns it green
(M2) — then proceeds as **thin vertical slices** (`Vertical A…F`): each slice
covers only the engine pieces its command needs plus the command itself, with its
acceptance suite — so a working command lands early and often instead of after a
long bottom-up engine phase. Within a slice, stories still follow build
dependency, and each story is built test-first (below). Each story has a stable,
milestone-prefixed **ID** (`M#`/`R#`/`P#`)
for reference in commits and task breakdowns, and an **area tag** (`[engine]`
core logic · `[cmd]` command · `[output]` UX/format · `[eng]` project/build
plumbing · `[build]` release/distribution · `[test]` tests · `[docs]`
documentation). Per [`testing-strategy.md`](./testing-strategy.md), **every**
story is **red/green / test-first with compiling red tests**: add any minimal
production compile seam needed for unit tests, write tests that compile and fail
for the expected behavior gap (unit tests for pure engine pieces, red acceptance
scripts in the slice's suite for commands), then implement to green — so test
work is folded into the stories it covers rather than listed as a separate phase.

**Status.** Stories use GitHub-style task markers: `[x]` complete, `[ ]`
pending. The next story is the first pending item in dependency order.

## MVP

_Goal: manage my own dotfiles daily on one machine — single source, both `home`
and `root` contexts, plan-by-default with `--apply`. Sequenced as **thin vertical
slices**: after the foundation, each slice delivers a working command end-to-end
(the engine pieces it needs + the command + its acceptance suite) rather than
building every engine primitive before the first command runs._

**Foundation**

1. [x] **M1** (red) Bootstrap just enough to fail a test first: `go mod init`, `mise` config (Go toolchain + task runner), and the `testscript` acceptance harness + unit scaffolding with the **per-suite layout** (build-tag–gated test hooks, isolated `$WORK`, `TUCK_TEST_STATE_DIR`, `readlink`/`wanthome` commands, and builtin `exec` / `! exec`). Land a **failing** first contract suite for `tuck --help` and framework-owned unknown-command behavior — see [`testing-strategy.md`](./testing-strategy.md) `[eng]`
2. [x] **M2** (green) Implement the `urfave/cli` command skeleton (global flags, subcommand stubs, and a main-style entrypoint registered with `testscript`) until the M1 contract suite passes `[eng]`

**Vertical A — Sources** (`source add` / `source list`)

3. [x] **M3** Read and validate a repository manifest (`<repo>/tuck.toml`: required `name`, optional `description`; ignore unknown keys) `[engine]`
4. [x] **M4** Discover and load machine-local source state (`${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml`; `TUCK_TEST_STATE_DIR` override) `[engine]`
5. [x] **M5** Validate machine state (unique enabled ids, ≤1 default, roots exist, **no overlapping roots**, readable manifests) `[engine]`
6. [x] **M6** Resolve the active source: `--source <id>` (id-only) → machine default → sole enabled source → `no_source` (exit 1, error.code `no_source`) `[engine]`
7. [x] **M7** Return meaningful error codes in JSON envelope and stderr (error.code values: `no_source`, `unknown_source`, `manifest_missing`, etc.) `[output]`
8. [x] **M8** Show actionable error messages with hints, on **stderr** `[output]`
9. [x] **M9** `source add <path> [--default]` — read manifest, atomic machine-state write, id-collision handling `[cmd]`
10. [x] **M10** `source list` — list enabled sources (id / path / default) `[cmd]`

**Vertical B — Package Use (home)**

11. [x] **M11** Parse and validate a plain package reference `[engine]`
12. [x] **M12** Convert package paths to target paths (and back), path-segment-aware `[engine]`
13. [x] **M13** Enumerate a package's leaf and directory entries (skip the reserved `.root` dir and `tuck.toml`) `[engine]`
14. [x] **M14** Resolve an existing package in the active source and context `[engine]`
15. [x] **M15** Classify a target path (absent / real file / dir / symlink / managed) `[engine]`
16. [x] **M16** Detect package-use and directory conflicts `[engine]`
17. [x] **M17** Build a complete action plan before any mutation `[engine]`
18. [x] **M18** Render a human-readable plan (plan / conflicts / summary) `[output]`
19. [x] **M19** Apply a conflict-free plan only when `--apply` is given (dry-run by default) `[engine]`
20. [x] **M20** `package use` a package's entries into the target tree (plan + `--apply`) `[cmd]`

**Vertical C — Status**

21. [x] **M21** Infer the owning package of a managed symlink **in the active source only** `[engine]`
22. [x] **M22** `package status` of a package's entries; `status` of a single target path (active-source ownership) `[cmd]`

**Vertical D — Package Drop**

23. [ ] **M23** `package drop` a package's managed symlinks `[cmd]`

**Vertical E — Adopt / eject**

24. [ ] **M24** `adopt` a real file into a package and link it back `[cmd]`
25. [ ] **M25** `eject` a managed file back to its target location (active-source ownership; `--source` valid) `[cmd]`

**Vertical F — Root context**

26. [ ] **M26** Operate in the `root` context (`--root`, package base `.root`, target `/`) with preflight privilege; error.code `privilege_required`; never self-escalate `[cmd]`
27. [ ] **M27** Root-context tests via the physical-root seam (`TUCK_TEST_ROOT_DIR`) with logical-path goldens; deterministic privilege tests via the injected predicate (`TUCK_TEST_PRIVILEGE`), covering `privilege_required` vs `io_error` `[test]`
28. [ ] **M28** Build and run on the developer's platform `[eng]`

<!-- ===================== MVP CUTOFF ===================== -->

---

## First Release

_Goal: a shippable v1 others can install and trust — machine output (JSON),
distributed builds, and docs._

1. [ ] **R1** `package refresh` a package (refresh + normalize link payloads) `[cmd]`
2. [x] **R2** List packages in the active source (`package list`) `[cmd]`
3. [ ] **R3** Show a package's file tree, and all packages in the active source (`package show`) `[cmd]`
4. [ ] **R4** `source rm <id>` — remove a source from machine state `[cmd]`
5. [ ] **R5** `source init <path> [--name <id>] [--description <text>]` — scaffold a repo `tuck.toml` manifest without registering it `[cmd]`
6. [ ] **R6** `source add <path> --init [--name <id>] [--description <text>]` — explicitly create a missing manifest, then register the source `[cmd]`
7. [ ] **R7** Interactive first-run init on `no_source`: prompt for a repo path and run `source add --init` when needed (TTY only; non-interactive still errors) `[cmd]`
8. [ ] **R8** Print global and per-command help/usage text `[output]`
9. [ ] **R9** Generate structured JSON help/usage metadata for root and per-command `--help --json` `[output]`
10. [ ] **R10** Expose the error classification system (`error.code` in JSON envelope and stderr) `[output]`
11. [ ] **R11** Report `multiple_providers` / `mismatch` / `owned_by_other` in `package status` `[cmd]`
12. [ ] **R12** Emit stable, versioned JSON for every command, including `source` (`--json`) `[output]`
13. [ ] **R13** JSON golden tests for every envelope `kind` (plan/packages/tree/status/sources/help/version/error) `[test]`
14. [ ] **R14** Parse repo-level package/file metadata in `tuck.toml` (`[package.<name>]`, `[[package.<name>.file]]`, `deploy`, `mode`) `[engine]`
15. [ ] **R15** Support `deploy = "copy"` package leaves: plan/apply copy actions, preserve explicit modes, and reject unsafe overwrites `[engine]`
16. [ ] **R16** Track copied-file deployments in machine-local state and report copy drift in `status` / `package status` `[engine]`
17. [ ] **R17** Validate text machine state with a generated checksum sidecar and report `state_checksum_mismatch` with repair guidance `[engine]`
18. [ ] **R18** Add JSON and human output coverage for copy actions, copy drift, explicit modes, and state checksum failures `[test]`
19. [ ] **R19** Auto-detect color; disable with `--no-color` (implied by `--json`) `[output]`
20. [ ] **R20** Acceptance coverage for each error code and each conflict rule `[test]`
21. [ ] **R21** Configure reproducible release builds with version stamping; maintain a changelog and `--version` output `[build]`
22. [ ] **R22** Run CI on PRs (build, unit + acceptance tests, vet) `[build]`
23. [ ] **R23** Enforce lint/format gates in CI `[build]`
24. [ ] **R24** Produce cross-platform / cross-arch binaries (linux+macos, amd64+arm64) `[build]`
25. [ ] **R25** Publish releases with attached binaries and checksums (GitHub Releases) `[build]`
26. [ ] **R26** Provide an install script and/or package-manager tap (e.g. Homebrew) `[build]`
27. [ ] **R27** Write a README with installation and quickstart `[docs]`
28. [ ] **R28** Publish a documentation website (spec, guides, worked examples) `[docs]`

<!-- ================= FIRST RELEASE CUTOFF ================= -->

---

## Post-Release / Future

_Unordered idea bucket; IDs are for reference only._

1. [ ] **P1** Add a `prune` / `doctor` command (empty dirs, mismatch diagnosis) `[cmd]`
2. [ ] **P2** Support nested package names `[engine]`
3. [ ] **P3** Allow user-configurable contexts beyond `home`/`root` `[engine]`
4. [ ] **P4** Make the apply/plan default configurable (machine-local preference) and reintroduce `--dry-run` `[cmd]`
5. [ ] **P5** Generate shell completions (bash/zsh/fish) `[cmd]`
6. [ ] **P6** Generate man pages `[docs]`
7. [ ] **P7** Provide an `adopt`-on-conflict shortcut from a `package use` conflict `[cmd]`
8. [ ] **P8** Re-introduce verbose/quiet output modes if needed `[output]`
9. [ ] **P9** Explore Windows support `[eng]`
10. [ ] **P10** Explore a watch / auto-refresh mode `[cmd]`
11. [ ] **P11** Add a machine-local source-id override (e.g. `source add --id <id>`) to resolve manifest-name collisions between repos `[cmd]`
12. [ ] **P12** Optionally prune now-empty intermediate directories left behind after `package drop` / `eject` `[cmd]`
13. [ ] **P13** Explore a boilerplate generator tool for const sentinel errors `[eng]`
14. [ ] **P14** Explore per-file hardlink deployment for symlink-hostile files on the same filesystem `[engine]`
