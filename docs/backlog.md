# tuck — User Story Backlog

Stories are grouped by milestone. Horizontal rules mark the **MVP cutoff** and
the **First Release cutoff**. Derived from [`cli-spec.md`](./cli-spec.md).

## MVP

_Goal: manage my own dotfiles daily on one machine — single source, both `home`
and `root` contexts, plan-by-default with `--apply`._

### Core engine
- Discover the config file (`--config` / `$TUCK_CONFIG` / XDG default)
- Parse the TOML config (sources, `[defaults]`)
- Validate config at startup (>=1 enabled source, unique ids, roots exist)
- Resolve the active source (single enabled source / `[defaults].source`)
- Parse and validate a plain package reference
- Resolve an existing package in the active source and context
- Enumerate a package's leaf and directory entries
- Convert package paths to target paths (and back), path-segment-aware
- Classify a target path (absent / real file / dir / symlink / managed)
- Infer the owning package of a managed symlink
- Detect deploy and directory conflicts
- Build a complete action plan before any mutation
- Apply a conflict-free plan only when `--apply` is given

### Commands
- Deploy a package's entries into the target tree
- Undeploy a package's managed symlinks
- Adopt a real file into a package and link it back
- Eject a managed file back to its target location
- Show status of a package's entries
- Show status of a single target path (`--path`)
- Operate in either context: `home` (target `$HOME`, default) or `root` (`--root`, package base `.root`, target `/`)
- Surface required privilege as a preflight check; exit `5`; never self-escalate

### Output & safety
- Render a human-readable plan (plan / conflicts / summary)
- Dry-run by default; mutate only with `--apply`
- Return meaningful exit codes (ok / conflict / usage / config / privilege / runtime)
- Show actionable error messages with hints

### Engineering
- Scaffold the Go module and command framework
- Unit-test path primitives, ownership inference, and conflict rules
- Stand up the `testscript` acceptance harness (build-tag–gated test hooks, isolated `$WORK` filetree, `readlink`/`wantexit` custom commands) — see [`testing-strategy.md`](./testing-strategy.md)
- Red/green acceptance tests for home- and root-context verbs (plan-by-default + `--apply`, exit codes, `readlink` payloads)
- Root-context tests via the physical-root seam (`TUCK_TEST_ROOT_DIR`) with logical-path goldens; deterministic privilege tests via the injected predicate (`TUCK_TEST_PRIVILEGE`), covering exit 5 vs exit 6
- Build and run on the developer's platform

<!-- ===================== MVP CUTOFF ===================== -->

---

## First Release

_Goal: a shippable v1 others can install and trust — machine output (JSON),
distributed builds, and docs._

### Feature completeness
- Redeploy a package (refresh + normalize link payloads)
- List packages in the active source (`packages`)
- Show a package's file tree, and all packages (`tree`)
- Emit stable, versioned JSON for every command (`--json`)
- Auto-detect color; disable with `--no-color` (implied by `--json`)
- Report `multiple_providers` / `mismatch` / `owned_by_other` in status
- Print global and per-command help/usage text
- Expose the full exit-code taxonomy and structured error envelope

### Build & distribution
- Configure reproducible release builds with version stamping
- Produce cross-platform / cross-arch binaries (linux+macos, amd64+arm64)
- Run CI on PRs (build, unit + acceptance tests, vet)
- Enforce lint/format gates in CI
- Publish releases with attached binaries and checksums (GitHub Releases)
- Provide an install script and/or package-manager tap (e.g. Homebrew)
- Maintain a changelog and `--version` output

### Testing
- JSON golden tests for every envelope `kind` (plan/packages/tree/status/error)
- Acceptance coverage for each non-zero exit code and each conflict rule

### Documentation
- Write a README with installation and quickstart
- Publish a documentation website (spec, guides, worked examples)

<!-- ================= FIRST RELEASE CUTOFF ================= -->

---

## Post-Release / Future

- Operate across multiple sources: select with `--source` and configure `[defaults].source`
- List and tree across all enabled sources (cross-source `packages` / `tree`)
- Add a `prune` / `doctor` command (empty dirs, mismatch diagnosis)
- Support nested package names
- Allow user-configurable contexts beyond `home`/`root`
- Make the apply/plan default configurable (`[defaults].apply`) and reintroduce `--dry-run`
- Add a config `init` command to scaffold `config.toml`
- Generate shell completions (bash/zsh/fish)
- Generate man pages
- Provide an `adopt`-on-conflict shortcut from a `deploy` conflict
- Add commands to enable/disable sources
- Re-introduce verbose/quiet output modes if needed
- Explore Windows support
- Explore a watch / auto-redeploy mode
