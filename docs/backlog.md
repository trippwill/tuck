# tuck — User Story Backlog

Stories are grouped by milestone. Horizontal rules mark the **MVP cutoff** and
the **First Release cutoff**. Derived from [`cli-spec.md`](./cli-spec.md).

## MVP

_Goal: manage my own dotfiles daily on one machine — single source, `home`
context, plan-by-default with `--apply`._

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

### Commands (home context)
- Deploy a package's entries into `$HOME`
- Undeploy a package's managed symlinks
- Adopt a real file into a package and link it back
- Eject a managed file back to its target location
- Show status of a package's entries
- Show status of a single target path (`--path`)

### Output & safety
- Render a human-readable plan (plan / conflicts / summary)
- Dry-run by default; mutate only with `--apply`
- Return meaningful exit codes (ok / conflict / usage / config / runtime)
- Show actionable error messages with hints

### Engineering
- Scaffold the Go module and command framework
- Unit-test path primitives, ownership inference, and conflict rules
- Build and run on the developer's platform

<!-- ===================== MVP CUTOFF ===================== -->

---

## First Release

_Goal: a shippable v1 others can install and trust — multi-source, `root`
context, machine output, distributed builds, and docs._

### Feature completeness
- Redeploy a package (refresh + normalize link payloads)
- List packages across all sources (`packages`)
- Show a package's file tree, and all packages (`tree`)
- Select among multiple sources with `--source`
- Configure a default source (`[defaults].source`) across many sources
- Operate in the `root` context via `--root` (target `/`)
- Surface required privilege, exit 5, never self-escalate
- Emit stable, versioned JSON for every command (`--json`)
- Auto-detect color; disable with `--no-color` (implied by `--json`)
- Report `multiple_providers` / `mismatch` / `owned_by_other` in status
- Print global and per-command help/usage text
- Expose the full exit-code taxonomy and structured error envelope

### Build & distribution
- Configure reproducible release builds with version stamping
- Produce cross-platform / cross-arch binaries (linux+macos, amd64+arm64)
- Run CI on PRs (build, test, vet)
- Enforce lint/format gates in CI
- Publish releases with attached binaries and checksums (GitHub Releases)
- Provide an install script and/or package-manager tap (e.g. Homebrew)
- Maintain a changelog and `--version` output

### Documentation
- Write a README with installation and quickstart
- Publish a documentation website (spec, guides, worked examples)

<!-- ================= FIRST RELEASE CUTOFF ================= -->

---

## Post-Release / Future

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
