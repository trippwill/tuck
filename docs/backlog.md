# tuck — User Story Backlog

A dependency-ordered backlog. Work top-to-bottom: the first unstarted story is
always the next one you can build. Stories are grouped into three milestones;
horizontal rules mark the **MVP cutoff** and the **First Release cutoff**.
Derived from [`cli-spec.md`](./cli-spec.md).

**Ordering.** Within each milestone, stories are sequenced by build dependency —
foundational and pure pieces first, then the code that consumes them, then
commands, then the `root` context. Each story has a stable, milestone-prefixed
**ID** (`M#`/`R#`/`P#`) for reference in commits and task breakdowns, and an
**area tag** (`[engine]` core logic · `[cmd]` command · `[output]` UX/format ·
`[eng]` project/build plumbing · `[build]` release/distribution · `[test]`
tests · `[docs]` documentation). Per [`testing-strategy.md`](./testing-strategy.md),
every engine and command story lands **test-first**: pure engine pieces carry
unit tests, and commands carry red/green acceptance scripts written before their
implementation — so test work is folded into the stories it covers rather than
listed as a separate phase.

## MVP

_Goal: manage my own dotfiles daily on one machine — single source, both `home`
and `root` contexts, plan-by-default with `--apply`._

1. **M1** Scaffold the Go module and command framework `[eng]`
2. **M2** Stand up the `testscript` acceptance harness + unit-test scaffolding (build-tag–gated test hooks, isolated `$WORK` filetree, `readlink`/`wantexit` custom commands) — see [`testing-strategy.md`](./testing-strategy.md) `[eng]`
3. **M3** Discover the config file (`--config` / `$TUCK_CONFIG` / XDG default) `[engine]`
4. **M4** Parse the TOML config (sources, `[defaults]`) `[engine]`
5. **M5** Validate config at startup (≥1 enabled source, unique ids, roots exist) `[engine]`
6. **M6** Resolve the active source (sole enabled source / `[defaults].source`) `[engine]`
7. **M7** Parse and validate a plain package reference `[engine]`
8. **M8** Convert package paths to target paths (and back), path-segment-aware `[engine]`
9. **M9** Enumerate a package's leaf and directory entries `[engine]`
10. **M10** Resolve an existing package in the active source and context `[engine]`
11. **M11** Classify a target path (absent / real file / dir / symlink / managed) `[engine]`
12. **M12** Infer the owning package of a managed symlink `[engine]`
13. **M13** Detect deploy and directory conflicts `[engine]`
14. **M14** Build a complete action plan before any mutation `[engine]`
15. **M15** Render a human-readable plan (plan / conflicts / summary) `[output]`
16. **M16** Return meaningful exit codes (ok / conflict / usage / config / privilege / runtime) `[output]`
17. **M17** Show actionable error messages with hints `[output]`
18. **M18** Apply a conflict-free plan only when `--apply` is given (dry-run by default) `[engine]`
19. **M19** `deploy` a package's entries into the target tree `[cmd]`
20. **M20** `status` of a package's entries and of a single target path (`--path`) `[cmd]`
21. **M21** `undeploy` a package's managed symlinks `[cmd]`
22. **M22** `adopt` a real file into a package and link it back `[cmd]`
23. **M23** `eject` a managed file back to its target location `[cmd]`
24. **M24** Operate in the `root` context (`--root`, package base `.root`, target `/`) with preflight privilege; exit `5`; never self-escalate `[cmd]`
25. **M25** Root-context tests via the physical-root seam (`TUCK_TEST_ROOT_DIR`) with logical-path goldens; deterministic privilege tests via the injected predicate (`TUCK_TEST_PRIVILEGE`), covering exit 5 vs exit 6 `[test]`
26. **M26** Build and run on the developer's platform `[eng]`

<!-- ===================== MVP CUTOFF ===================== -->

---

## First Release

_Goal: a shippable v1 others can install and trust — machine output (JSON),
distributed builds, and docs._

1. **R1** Redeploy a package (refresh + normalize link payloads) `[cmd]`
2. **R2** List packages in the active source (`packages`) `[cmd]`
3. **R3** Show a package's file tree, and all packages (`tree`) `[cmd]`
4. **R4** Print global and per-command help/usage text `[output]`
5. **R5** Expose the full exit-code taxonomy and structured error envelope `[output]`
6. **R6** Report `multiple_providers` / `mismatch` / `owned_by_other` in status `[cmd]`
7. **R7** Emit stable, versioned JSON for every command (`--json`) `[output]`
8. **R8** JSON golden tests for every envelope `kind` (plan/packages/tree/status/error) `[test]`
9. **R9** Auto-detect color; disable with `--no-color` (implied by `--json`) `[output]`
10. **R10** Acceptance coverage for each non-zero exit code and each conflict rule `[test]`
11. **R11** Configure reproducible release builds with version stamping; maintain a changelog and `--version` output `[build]`
12. **R12** Run CI on PRs (build, unit + acceptance tests, vet) `[build]`
13. **R13** Enforce lint/format gates in CI `[build]`
14. **R14** Produce cross-platform / cross-arch binaries (linux+macos, amd64+arm64) `[build]`
15. **R15** Publish releases with attached binaries and checksums (GitHub Releases) `[build]`
16. **R16** Provide an install script and/or package-manager tap (e.g. Homebrew) `[build]`
17. **R17** Write a README with installation and quickstart `[docs]`
18. **R18** Publish a documentation website (spec, guides, worked examples) `[docs]`

<!-- ================= FIRST RELEASE CUTOFF ================= -->

---

## Post-Release / Future

_Unordered idea bucket; IDs are for reference only._

1. **P1** Operate across multiple sources: select with `--source` and configure `[defaults].source`; list/tree across all enabled sources `[cmd]`
2. **P2** Add a `prune` / `doctor` command (empty dirs, mismatch diagnosis) `[cmd]`
3. **P3** Support nested package names `[engine]`
4. **P4** Allow user-configurable contexts beyond `home`/`root` `[engine]`
5. **P5** Make the apply/plan default configurable (`[defaults].apply`) and reintroduce `--dry-run` `[cmd]`
6. **P6** Add a config `init` command to scaffold `config.toml` `[cmd]`
7. **P7** Generate shell completions (bash/zsh/fish) `[cmd]`
8. **P8** Generate man pages `[docs]`
9. **P9** Provide an `adopt`-on-conflict shortcut from a `deploy` conflict `[cmd]`
10. **P10** Add commands to enable/disable sources `[cmd]`
11. **P11** Re-introduce verbose/quiet output modes if needed `[output]`
12. **P12** Explore Windows support `[eng]`
13. **P13** Explore a watch / auto-redeploy mode `[cmd]`
