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
story is **red/green / test-first**: write the failing test (unit for pure engine
pieces, a red acceptance script in the slice's suite for commands), then
implement to green — so test work is folded into the stories it covers rather than
listed as a separate phase.

## MVP

_Goal: manage my own dotfiles daily on one machine — single source, both `home`
and `root` contexts, plan-by-default with `--apply`. Sequenced as **thin vertical
slices**: after the foundation, each slice delivers a working command end-to-end
(the engine pieces it needs + the command + its acceptance suite) rather than
building every engine primitive before the first command runs._

**Foundation**

1. **M1** (red) Bootstrap just enough to fail a test first: `go mod init`, `mise` config (Go toolchain + task runner), and the `testscript` acceptance harness + unit scaffolding with the **per-suite layout** (build-tag–gated test hooks, isolated `$WORK`, `TUCK_TEST_STATE_DIR`, `readlink`/`wantexit`/`wanthome` commands). Land a **failing** first contract suite: `tuck --help` exits `0`, an unknown command exits `2` — see [`testing-strategy.md`](./testing-strategy.md) `[eng]`
2. **M2** (green) Implement the `urfave/cli` command skeleton (global flags, subcommand stubs, a testable `Run(args, env, …) int` core) until the M1 contract suite passes `[eng]`

**Vertical A — Sources** (`source enable` / `source list`)

3. **M3** Read and validate a repository manifest (`<repo>/tuck.toml`: required `name`, optional `description`; ignore unknown keys) `[engine]`
4. **M4** Discover and load machine-local source state (`${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml`; `TUCK_TEST_STATE_DIR` override) `[engine]`
5. **M5** Validate machine state (unique enabled ids, ≤1 default, roots exist, **no overlapping roots**, readable manifests) `[engine]`
6. **M6** Resolve the active source: `--source <id>` (id-only) → machine default → sole enabled source → `no_source` (exit 3) `[engine]`
7. **M7** Return meaningful exit codes (ok / conflict / usage / config-state / resolution / privilege / runtime) `[output]`
8. **M8** Show actionable error messages with hints, on **stderr** `[output]`
9. **M9** `source enable <path> [--default]` — read manifest, atomic machine-state write, id-collision handling `[cmd]`
10. **M10** `source list` — list enabled sources (id / path / default) `[cmd]`

**Vertical B — Deploy (home)**

11. **M11** Parse and validate a plain package reference `[engine]`
12. **M12** Convert package paths to target paths (and back), path-segment-aware `[engine]`
13. **M13** Enumerate a package's leaf and directory entries (skip the reserved `.root` dir and `tuck.toml`) `[engine]`
14. **M14** Resolve an existing package in the active source and context `[engine]`
15. **M15** Classify a target path (absent / real file / dir / symlink / managed) `[engine]`
16. **M16** Detect deploy and directory conflicts `[engine]`
17. **M17** Build a complete action plan before any mutation `[engine]`
18. **M18** Render a human-readable plan (plan / conflicts / summary) `[output]`
19. **M19** Apply a conflict-free plan only when `--apply` is given (dry-run by default) `[engine]`
20. **M20** `deploy` a package's entries into the target tree (plan + `--apply`) `[cmd]`

**Vertical C — Status**

21. **M21** Infer the owning package of a managed symlink **in the active source only** `[engine]`
22. **M22** `status` of a package's entries and of a single target path (`--path`, active-source ownership) `[cmd]`

**Vertical D — Undeploy**

23. **M23** `undeploy` a package's managed symlinks `[cmd]`

**Vertical E — Adopt / eject**

24. **M24** `adopt` a real file into a package and link it back `[cmd]`
25. **M25** `eject` a managed file back to its target location (active-source ownership; `--source` valid) `[cmd]`

**Vertical F — Root context**

26. **M26** Operate in the `root` context (`--root`, package base `.root`, target `/`) with preflight privilege; exit `5`; never self-escalate `[cmd]`
27. **M27** Root-context tests via the physical-root seam (`TUCK_TEST_ROOT_DIR`) with logical-path goldens; deterministic privilege tests via the injected predicate (`TUCK_TEST_PRIVILEGE`), covering exit 5 vs exit 6 `[test]`
28. **M28** Build and run on the developer's platform `[eng]`

<!-- ===================== MVP CUTOFF ===================== -->

---

## First Release

_Goal: a shippable v1 others can install and trust — machine output (JSON),
distributed builds, and docs._

1. **R1** Redeploy a package (refresh + normalize link payloads) `[cmd]`
2. **R2** List packages in the active source (`packages`) `[cmd]`
3. **R3** Show a package's file tree, and all packages in the active source (`tree`) `[cmd]`
4. **R4** `source disable <id>` — disable a source in machine state `[cmd]`
5. **R5** Interactive first-run init on `no_source`: prompt for a repo path and run `source enable` (TTY only; non-interactive still errors) `[cmd]`
6. **R6** Print global and per-command help/usage text `[output]`
7. **R7** Expose the full exit-code taxonomy and structured error envelope `[output]`
8. **R8** Report `multiple_providers` / `mismatch` / `owned_by_other` in status `[cmd]`
9. **R9** Emit stable, versioned JSON for every command, including `source` (`--json`) `[output]`
10. **R10** JSON golden tests for every envelope `kind` (plan/packages/tree/status/sources/error) `[test]`
11. **R11** Auto-detect color; disable with `--no-color` (implied by `--json`) `[output]`
12. **R12** Acceptance coverage for each non-zero exit code and each conflict rule `[test]`
13. **R13** Configure reproducible release builds with version stamping; maintain a changelog and `--version` output `[build]`
14. **R14** Run CI on PRs (build, unit + acceptance tests, vet) `[build]`
15. **R15** Enforce lint/format gates in CI `[build]`
16. **R16** Produce cross-platform / cross-arch binaries (linux+macos, amd64+arm64) `[build]`
17. **R17** Publish releases with attached binaries and checksums (GitHub Releases) `[build]`
18. **R18** Provide an install script and/or package-manager tap (e.g. Homebrew) `[build]`
19. **R19** Write a README with installation and quickstart `[docs]`
20. **R20** Publish a documentation website (spec, guides, worked examples) `[docs]`

<!-- ================= FIRST RELEASE CUTOFF ================= -->

---

## Post-Release / Future

_Unordered idea bucket; IDs are for reference only._

1. **P1** Add a `prune` / `doctor` command (empty dirs, mismatch diagnosis) `[cmd]`
2. **P2** Support nested package names `[engine]`
3. **P3** Allow user-configurable contexts beyond `home`/`root` `[engine]`
4. **P4** Make the apply/plan default configurable (machine-local preference) and reintroduce `--dry-run` `[cmd]`
5. **P5** Add a `source init` command to scaffold a repo `tuck.toml` manifest `[cmd]`
6. **P6** Generate shell completions (bash/zsh/fish) `[cmd]`
7. **P7** Generate man pages `[docs]`
8. **P8** Provide an `adopt`-on-conflict shortcut from a `deploy` conflict `[cmd]`
9. **P9** Re-introduce verbose/quiet output modes if needed `[output]`
10. **P10** Explore Windows support `[eng]`
11. **P11** Explore a watch / auto-redeploy mode `[cmd]`
12. **P12** Add a machine-local source-id override (e.g. `source enable --id <id>`) to resolve manifest-name collisions between repos `[cmd]`
13. **P13** Optionally prune now-empty intermediate directories left behind after `undeploy` / `eject` `[cmd]`
