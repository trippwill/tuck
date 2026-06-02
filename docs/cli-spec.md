# `tuck` CLI — Implementation Specification

Status: authoritative for the **command-line interface**.
Scope: command surface, flags, configuration, output (human + JSON), exit codes,
error reporting, and plan rendering.

This document is the source of truth for the CLI **and** its internal resolution
algorithms — it is fully self-contained. [§12](#12-resolution-algorithms)
specifies the path primitives, source/package resolution, package-entry
enumeration, ownership inference, conflict rules, and operation algorithms
normatively. It supersedes an earlier exploratory draft
(`resolution-algorithms.md`, since removed); see
[Appendix B](#appendix-b--relationship-to-the-draft) for the deliberate changes
made relative to that draft.

`tuck` is a Go dotfiles manager that replaces the legacy `d.*` zsh
aliases/functions. It maps package directories onto real target directories,
links only leaf entries, never folds directories, and never lets the caller's
working directory affect correctness (except when resolving an explicitly
relative input path).

---

## 1. Concepts

These are summarized here for CLI grounding. Full, normative definitions and
algorithms are in [§12](#12-resolution-algorithms).

- **Source** — a dotfiles repository. Each repository carries a committed
  manifest (`<repo>/tuck.toml`) declaring its short `name` (used as the source
  `id`). A source is made usable on a machine with `tuck source enable <path>`,
  which records its path and enabled state in machine-local state
  ([§5](#5-configuration), [§12.1](#121-contexts-bases-and-identities)). Every
  command operates on exactly **one active source** ([§3.2](#32-source-selection)).
- **Target context** — selects where package entries appear:
  - `home`: package base `<source.path>`, target root `$HOME` (default). (The
    earlier draft called this context `user`; see
    [Appendix B](#appendix-b--relationship-to-the-draft).)
  - `root`: package base `<source.path>/.root`, target root `/`.

  ([§12.1](#121-contexts-bases-and-identities))
- **Package** — one directory inside a package base. Identified internally as
  `source:context:name` (e.g. `public:home:zsh`); CLI input never includes the
  source or the context — both are selected by flags, not encoded in the ref.
  ([§12.1](#121-contexts-bases-and-identities))
- **Package reference (ref)** — CLI input naming a package by its plain
  `<name>` (e.g. `zsh`). The source is not part of the ref; it is selected
  separately (see [§3.2](#32-source-selection), [§6](#6-package-references)).
- **Managed entry** — a target symlink whose resolved destination is inside a
  configured package root at the matching package-relative path. Ownership is
  inferred from the link payload; there is no manifest.
  ([§12.5](#125-ownership-resolution))
- **Leaf vs. directory entries** — only leaf entries become symlinks; directory
  entries become real directories in the target tree.
  ([§12.4](#124-package-entry-enumeration))

---

## 2. Command surface

```text
tuck [global-flags] <command> [command-args] [command-flags]
```

Two verb sets, distinguished by **whether a real file ever moves**:

| Set | Commands | Moves real files? | Default execution |
| --- | --- | --- | --- |
| Symlink-only | `deploy`, `undeploy`, `redeploy` | No | dry-run; needs `--apply` |
| File movement | `adopt`, `eject` | Yes | dry-run; needs `--apply` |
| Read-only | `packages`, `tree`, `status` | No | n/a |
| Source management | `source enable`, `source list` | No | `enable` writes machine state immediately; `list` is read-only |

All five mutating verbs **plan by default and mutate only with `--apply`**; the
two sets differ only in what they touch (symlinks vs. real file bytes), not in
how they are confirmed.

Synopsis:

```text
tuck deploy   <package-ref>...
tuck undeploy <package-ref>...
tuck redeploy <package-ref>...
tuck adopt    <package-ref> <target-file>
tuck eject    <target-link>
tuck packages
tuck tree     [package-ref]
tuck status   [package-ref] [--path <target-path>]
tuck source   enable <path> [--default]
tuck source   list
```

Semantics:

- `deploy` — create managed target symlinks for a package's leaf entries.
- `undeploy` — remove managed target symlinks for selected packages. It does
  **not** move package files back into the target tree.
- `redeploy` — refresh selected package links (undeploy then deploy), also
  normalizing symlink payloads to the preferred relative form.
- `adopt` — move one existing **real** target file into a package, then create a
  managed symlink pointing back to it.
- `eject` — remove one managed target symlink and move the package file back
  into the target tree.
- `packages` — list the active source's package directories.
- `tree` — display package contents.
- `status` — report managed/conflicting/absent state for packages or a path.
- `source enable` — register and enable a dotfiles repository on this machine
  (reads its `tuck.toml`, writes machine-local state).
- `source list` — list the sources enabled on this machine.

> **Name mapping from the draft.** `link→deploy`, `unlink→undeploy`,
> `relink→redeploy`, `capture→adopt`, `release→eject`. The legacy `d.unlink`
> materialization workflow maps to **`eject`**, not `undeploy`. This rename
> removes the draft's documented footgun where "unlink" could be mistaken for
> "restore the real file".

There is no `tuck home` or `tuck root` command. `home` is the default context;
the `root` context is selected with the global `--root` flag
(see [§3.1](#31-context-selection)), not a command prefix.

---

## 3. Invocation model

```text
tuck [global-flags] <command> [args] [command-flags]
```

- Global flags may appear before the command or after it; both forms are
  accepted and equivalent. (`tuck --json deploy zsh` ≡ `tuck deploy zsh --json`.)
- `--` terminates flag parsing; subsequent tokens are positional arguments.
- Unknown commands and unknown flags are usage errors (exit `2`).
- With no command, `tuck` prints top-level help and exits `0`.
- `--help`, `--version`, and the no-command case **bypass** machine-state
  discovery and source resolution entirely; they never require any enabled
  source and always succeed (exit `0`). They also ignore `--json` and always
  print plain text.
- `tuck source enable <path>` does **not** require a pre-existing active source —
  it establishes one. `tuck source list` requires only readable machine state
  (an empty/absent state is reported as "no sources enabled", not an error).

### 3.1 Context selection

There are exactly two target contexts. `home` is the unconditional default; the
boolean `--root` flag selects the `root` context instead:

- No flag → `home` (package base `<source.path>`, target root `$HOME`).
- `--root` → `root` (package base `<source.path>/.root`, target root `/`).

`--root` is idempotent: repeating it has no additional effect. There is no
`--home` flag and no config setting for the default context — `home` is always
the default unless `--root` is given.

### 3.2 Source selection

Every command operates on exactly **one active source**. The active source is
resolved in this order (first wins):

1. `--source <id>` (`-s`) on the command line.
2. The machine-local **default** source (set with `tuck source enable --default`).
3. The sole enabled source, when exactly one source is enabled.
4. Otherwise → error `no_source` (exit `3`):
   `no source selected; run \`tuck source enable <path>\` or pass --source`.

`--source` accepts a **source id only**. The value must name an **enabled**
source in machine-local state ([§5.3](#53-machine-local-state)); an unknown or
disabled id is `unknown_source` (exit `4`). A source must be enabled with
`tuck source enable` ([§7.9](#79-source)) before it can be selected — there is
no ad-hoc, path-based source selection.

Package refs are plain names and never encode a source (see
[§6](#6-package-references)); a package that does not exist in the active source
is `package_not_found` (exit `4`) — there is **no cross-source search**. To
operate on a different source, select it with `--source` (or change the default)
and run the command again.

There is **no simultaneous multi-source operation**. The listing commands
`packages` and `tree` (without a ref) list only the **active source**. The
path-based commands `eject` and `status --path` infer ownership in the **active
source only** ([§12.5](#125-ownership-resolution)); `--source` is valid on them.
A managed symlink that points into a non-active source reads as `unmanaged`
unless that source is made active.

---

## 4. Global flags

These apply to every command unless noted.

| Flag | Alias | Argument | Default | Meaning |
| --- | --- | --- | --- | --- |
| `--root` | | — | off | Select the `root` context (target root `/`); default context is `home`. |
| `--source` | `-s` | id | §3.2 | Select the active source by enabled id. |
| `--json` | | — | off | Emit a single JSON document; suppress human output. |
| `--apply` | | — | off | Execute the plan: perform mutations for `deploy`/`undeploy`/`redeploy`/`adopt`/`eject`. Without it, mutating verbs only print the plan. |
| `--no-color` | | — | off | Disable ANSI color in human output (otherwise color is auto-enabled on a TTY). |
| `--version` | `-V` | — | — | Print version and exit `0`. |
| `--help` | `-h` | — | — | Print help for the program or command and exit `0`. |

Flag interaction rules:

- Mutating verbs build a plan and **print it without mutating** unless `--apply`
  is given; this dry-run-by-default is the only no-op mode (there is no
  `--dry-run` flag — see [Appendix B](#appendix-b--relationship-to-the-draft)).
- `--json` implies `--no-color` and emits the machine payload instead of human
  output.
- `--source` selects the single active source by enabled id
  (see [§3.2](#32-source-selection)). It is valid on every command, including
  `eject` and `status --path`.

---

## 5. Configuration

`tuck` has **no central config file**. Configuration is split between two
artifacts, matching what is portable versus what is machine-specific:

- a **repository manifest** committed in each dotfiles repo (`<repo>/tuck.toml`),
  which carries the repo's portable identity ([§5.2](#52-repository-manifest));
- **machine-local state** (`sources.toml`), generated by the `tuck source`
  commands, which records which repos are enabled on *this* machine and where
  they live ([§5.3](#53-machine-local-state)).

This split dissolves the bootstrap loop that a repo-managed central config would
create (the file that locates the repo cannot itself live in the repo it
locates): on a new machine the only machine-specific fact you supply is the
clone path, via `tuck source enable <path>`.

There is no `--config` flag and no `$TUCK_CONFIG`; both are removed. There is no
config setting for the default context; `home` is always the default unless
`--root` is passed (see [§3.1](#31-context-selection)).

### 5.1 Bootstrap

A new machine needs three steps; the loop is dissolved because `tuck source
enable` takes the clone **path** explicitly:

```text
git clone <repo> ~/.dotfiles
tuck source enable ~/.dotfiles        # reads ~/.dotfiles/tuck.toml, writes machine state
tuck deploy <packages> --apply
```

`tuck.toml` is a **file** at the repository root, not a package directory, so it
is ignored by package enumeration ([§12.8](#128-listing-algorithms)).

### 5.2 Repository manifest

Committed in the repo at `<repo>/tuck.toml`. Portable across machines because it
describes the repo's identity, not any machine's paths. TOML:

```toml
name        = "public"            # required: short source id / display identity
description = "public dotfiles"   # optional
```

Fields:

- `name` (required) — the repo's short id. Used as the source id and in display
  identities ([§12.1](#121-contexts-bases-and-identities)). Must not be empty and
  must not contain a path separator or `:`.
- `description` (optional) — a human-readable label shown in `source list`.

The format is **open to additive keys**; a future `[security]` block is
reserved. Unknown top-level keys are ignored so that newer repos remain readable
by older binaries. A missing or unreadable `tuck.toml` is `manifest_missing`
(exit `3`); a malformed one, or one missing a valid `name`, is `manifest_invalid`
(exit `3`).

### 5.3 Machine-local state

Generated and updated by `tuck source enable` ([§7.9](#79-source)); **not**
committed. Location:

```text
${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml
```

(For tests, the state directory may be overridden by `TUCK_TEST_STATE_DIR`, which
is compiled in only under the `tuck_testhooks` build tag and is absent from
release builds — see `docs/testing-strategy.md`.) TOML:

```toml
[[source]]
path    = "/home/me/.dotfiles"        # canonical repository path on this machine
id      = "public"                    # effective id (the manifest name)
enabled = true
default = true                        # at most one source may be the default

[[source]]
path    = "/home/me/.dotfiles-private"
id      = "private"
enabled = true
```

Fields per `[[source]]` entry:

- `path` (required) — the canonical repository path on this machine.
- `id` (required) — the **effective** source id, authoritative for selection and
  display identities. In MVP it is always the manifest `name`; a machine-local
  id override (to resolve collisions) is a Post-Release addition.
- `enabled` (optional, default `true`) — whether the source participates.
- `default` (optional, default `false`) — the machine-local default active
  source. Machine-local state always wins over anything a repo declares.

If the state file is absent or has no entries, no source is enabled; a command
that needs an active source fails with `no_source` (exit `3`) and a hint to run
`tuck source enable`. Reading the state for `source list` is not itself an error.

### 5.4 Validation

Performed when the state is loaded, before any command logic:

- Effective `id` values are **unique** across enabled entries and must not be
  empty or contain a path separator or `:`.
- At most one enabled entry has `default = true`, and a default entry must be
  enabled.
- Each enabled `path` is expanded and canonicalized
  ([§12.2](#122-path-primitives)); a path that does not exist is a state error.
- **Enabled source roots must not overlap:** no enabled `path` may equal,
  contain, or be contained by another enabled `path` (path-segment aware,
  [§12.2](#122-path-primitives)). This prevents one repo's files from being
  misattributed to another during ownership inference ([§12.5](#125-ownership-resolution)).
- Each enabled `path` must contain a readable, valid `tuck.toml`
  ([§5.2](#52-repository-manifest)).
- For the `root` context, the package base is `<source.path>/.root`. A source
  that has no `.root` directory contributes **no** packages in the root context
  but is not itself an error.

All state and manifest failures use exit code `3` and the `error` JSON envelope
when `--json` is set.

---

## 6. Package references

A package reference is a plain package name; the source is selected separately
(see [§3.2](#32-source-selection)) and is never part of the ref.

```text
package-ref := package-name
```

- Examples: `zsh`, `ssh`, `git`.
- A ref names a package within the **active source** and the active context.

Rejected (usage error, exit `2`):

- A ref containing `:`. (Source qualification via `source:name` is **not**
  supported; use `--source`. The `:` is reserved and rejected to avoid silent
  misuse.)
- Empty package name.
- Absolute package name, or a name containing `..` path segments.
- A package name containing a path separator. Packages are direct children of
  the package base; nested package names are reserved for a future revision and
  are rejected here so that ownership inference (which keys on the first path
  segment, [§12.5](#125-ownership-resolution)) cannot disagree with deployment.

Resolution of a ref to a concrete package (the algorithms in
[§12.3](#123-source-and-package-resolution) apply, with the source fixed to the
active source rather than searched):

- `deploy`, `undeploy`, `redeploy`, `tree <ref>`, `status <ref>` — the package
  must exist in the active source and context, else `package_not_found`
  (exit `4`).
- `adopt` — the package may already exist in the active source, or be created
  there if absent. Because the source is fixed, there is no "new package
  requires source qualification" case (see
  [§12.3 Resolve package for adopt](#123-source-and-package-resolution)).

---

## 7. Command reference

Each command below specifies: synopsis, arguments, flags, behavior, output,
errors, and exit codes. All mutating commands first build a **complete plan**;
if any conflict is found they print conflicts and mutate nothing
([§12.7](#127-operation-algorithms), [§12.9](#129-execution-planning)).

In addition to the per-command exit codes listed, **every** command may return
the global codes from [§10](#10-exit-codes): `2` (usage), `3` (config/state,
e.g. invalid machine state, a missing/invalid manifest, or `no_source`), and `4`
(resolution, e.g. an unknown `--source` id or a package not found). These are not
repeated in each command for brevity.

### 7.1 `deploy`

```text
tuck [--root] deploy <package-ref>...
```

- **Arguments:** one or more package names; each must exist in the active source and context.
- **Relevant flags:** `--apply`, `--json`, `--source`, `--root`.
- **Behavior:** for each resolved package, enumerate entries; plan `mkdir` for
  absent directory entries and `symlink` for linkable leaf entries; treat the
  rest as conflicts ([§12.7](#127-operation-algorithms)). A target produced by
  two different packages in the same invocation is a conflict
  (`multiple_providers`). Symlink payloads are relative
  (`relativePath(dirname(targetPath), packageEntryPath)`).
- **Execution:** **dry-run by default**; mutates only with `--apply` (and only
  when conflict-free).
- **Exit codes:** `0` applied/dry-run clean · `1` conflicts · `4` ref
  resolution · `5` privilege (root) · `6` runtime.

### 7.2 `undeploy`

```text
tuck [--root] undeploy <package-ref>...
```

- **Arguments:** one or more package names; each must exist in the active source and context.
- **Relevant flags:** `--apply`, `--json`, `--source`, `--root`.
- **Behavior:** for each leaf entry, plan `remove_symlink` only when the target
  is a symlink managed by the selected package for the same entry; absent
  targets are no-ops; anything else is a conflict / "not owned by selected
  package" ([§12.7](#127-operation-algorithms)). Directories are **never**
  pruned.
- **Execution:** **dry-run by default**; mutates only with `--apply` (and only
  when conflict-free).
- **Exit codes:** as `deploy`.

### 7.3 `redeploy`

```text
tuck [--root] redeploy <package-ref>...
```

- **Arguments:** one or more package names; each must exist in the active source and context.
- **Behavior:** build an undeploy plan, then a deploy plan against the
  post-undeploy state; if either has conflicts, stop. Apply removals first, then
  links ([§12.7](#127-operation-algorithms)). Already-owned links are removed
  and recreated to normalize the payload.
- **Relevant flags / execution / exit codes:** as `deploy` (dry-run by default;
  mutates only with `--apply`).

### 7.4 `adopt`

```text
tuck [--root] adopt <package-ref> <target-file>
```

- **Arguments:** one package name (in the active source; created there if it
  does not yet exist) and one existing **real** target file.
- **Relevant flags:** `--apply`, `--json`, `--source`,
  `--root`.
- **Behavior:** expand `<target-file>` without following the final symlink;
  reject if outside the target root or inside **any enabled source repository**;
  reject unless it classifies as a real file;
  convert to the destination package path in the active source; reject if that
  path already exists. Plan: `mkdir` parents → `move` target→package →
  `symlink` target→package ([§12.7](#127-operation-algorithms)).
- **Execution:** **dry-run by default**; mutates only with `--apply`.
- **Exit codes:** `0` applied or dry-run printed · `1` conflict · `2` usage ·
  `3` config/state (incl. `no_source`) · `4` resolution · `5` privilege ·
  `6` runtime.

### 7.5 `eject`

```text
tuck [--root] eject <target-link>
```

- **Arguments:** one target path that is a managed symlink. Its owner is inferred
  in the **active source** ([§3.2](#32-source-selection),
  [§12.5](#125-ownership-resolution)); a symlink that points into a non-active
  source reads as `unmanaged` and is rejected — point `--source` at the owning
  repo to eject it.
- **Relevant flags:** `--apply`, `--json`, `--source`, `--root`.
- **Behavior:** expand `<target-link>` without following the final symlink;
  classify in the selected context and active source; reject unless it is a
  managed symlink whose link path matches the package-relative target and whose
  package file exists and is not a directory. Plan: `remove_symlink` → `move`
  package→target ([§12.7](#127-operation-algorithms)). The now-empty package
  directory is left in place.
- **Execution:** **dry-run by default**; mutates only with `--apply`.
- **Exit codes:** `0` applied or dry-run printed · `1` conflict · `2` usage ·
  `3` config/state (incl. `no_source`) · `4` resolution · `5` privilege ·
  `6` runtime.

### 7.6 `packages`

```text
tuck [--root] packages
```

- **Behavior:** list the direct child package directories of the **active
  source's** package base ([§12.8](#128-listing-algorithms)). The reserved
  `.root` directory and the `tuck.toml` manifest are not packages and are
  omitted.
- **Flags:** `--source`, `--json`.
- **Exit codes:** `0` · `3` config/state (incl. `no_source`) · `4` unknown
  `--source`.

### 7.7 `tree`

```text
tuck [--root] tree [package-ref]
```

- **Behavior:** without a ref, show the **active source's** package tree grouped
  by package; with a ref, resolve it in the active source and show that package
  root's tree ([§12.8](#128-listing-algorithms)).
- **Flags:** `--source`, `--json`.
- **Exit codes:** `0` · `3` config/state (incl. `no_source`) · `4` unknown
  `--source` / package not found.

### 7.8 `status`

```text
tuck [--root] status [package-ref] [--path <target-path>]
```

- **Behavior:** read-only classification using
  [§12.5 Classify target path](#125-ownership-resolution).
  - With `--path <target-path>`: classify a single target path and report its
    state (and owner, if managed). The owner is inferred in the **active source**,
    so `--source` applies; a symlink owned by a non-active source reads as
    `unmanaged`.
  - With `<package-ref>`: resolve the package in the active source; for each leaf
    entry report one of: `deployed` (managed by this package, matching entry),
    `absent`, `conflict` (with reason), `mismatch` (managed-path mismatch), or
    `owned_by_other` (managed by a different package).
  - With neither: summarize every package in the active source and context
    (counts of deployed / absent / conflicting entries per package).
- **Flags:** `--path` (mutually exclusive with a package ref), `--source`,
  `--json`.
- **Exit codes:** `0` always for a successful read (the presence of conflicts is
  reported in the body, not via exit code) · `3` config/state (incl. `no_source`)
  · `4` unknown `--source` / not found.

Mapping from [§12.5 Classify target path](#125-ownership-resolution) to reported
`state`:

| Classification | `state` (with package ref) | `state` (`--path`, no package) |
| --- | --- | --- |
| `Absent` | `absent` | `absent` |
| `ManagedBySelectedPackage` (matching entry) | `deployed` | — |
| `ManagedSymlink` (any owner) | — | `deployed` |
| `ManagedByOtherPackage` | `owned_by_other` | `owned_by_other` |
| `ManagedPathMismatch` | `mismatch` | `mismatch` |
| `UnmanagedSymlink` | `conflict` (`unmanaged_symlink`) | `unmanaged` |
| `RealFile` | `conflict` (`real_file`) | `unmanaged` |
| `RealDirectory` | `conflict` (`real_directory`) | `unmanaged` |
| `SpecialFile` | `conflict` (`special_file`) | `unmanaged` |

With a package ref, `state` reflects whether each leaf entry can be/ is deployed
by that package, so non-owned real/unmanaged targets are `conflict` with the
matching `code`. With `--path` (no selected package), the same filesystem
states that are not managed are reported neutrally as `unmanaged`.

### 7.9 `source`

Manage which dotfiles repositories are enabled on this machine
([§5.3](#53-machine-local-state)). These commands operate on machine-local state,
not on the target tree; they do **not** select or require a pre-existing active
source.

```text
tuck source enable <path> [--default]
tuck source list
```

#### `source enable`

- **Arguments:** one repository `<path>` (expanded and canonicalized).
- **Flags:** `--default` (make this the machine-local default active source),
  `--json`.
- **Behavior:** read `<path>/tuck.toml` ([§5.2](#52-repository-manifest)); the
  effective id is the manifest `name`. Record
  (or update in place) the `[[source]]` entry in machine-local state with the
  canonical path, effective id, `enabled = true`, and `default` per `--default`.
  Setting `--default` clears any other entry's default. The write is validated
  against [§5.4](#54-validation) (unique ids, no overlapping roots, at most one
  default) and is **atomic**; on any validation failure nothing is written.
  Enabling a path already present is idempotent (updates the existing entry).
- **Exit codes:** `0` enabled · `2` usage · `3` config/state (missing/invalid
  manifest, id collision, overlapping root, invalid state) ·
  `6` runtime (state write failure).

Id collisions: if the effective id is already used by a **different** path,
`enable` fails (`state_invalid`, exit `3`). Resolving a collision with a
machine-local id override is a Post-Release feature; in MVP the two repos must
carry distinct manifest names.

#### `source list`

- **Behavior:** list the `[[source]]` entries from machine-local state — id,
  path, enabled, and whether each is the default — in declaration order. Absent
  or empty state reports "no sources enabled" and exits `0`.
- **Flags:** `--json`.
- **Exit codes:** `0` · `3` invalid state.

---

## 8. Plan and action model

Mutating commands emit an ordered list of **actions**
([§12.9](#129-execution-planning)):

| Action | Fields | Meaning |
| --- | --- | --- |
| `mkdir` | `path` | Create a real directory in the target tree. |
| `symlink` | `linkPath`, `payload`, `target` | Create a symlink; `payload` is the (relative) link text, `target` its resolved absolute destination. |
| `remove_symlink` | `path` | Remove a managed target symlink. |
| `move` | `src`, `dst` | Move a real file (adopt: target→package; eject: package→target). |

Planning rules:

1. Resolve all sources, packages, target paths, and ownership before mutating.
2. Accumulate **all** conflicts (do not stop at the first).
3. If any conflict exists: print conflicts, print no actions as applied, exit
   `1`. Nothing is mutated.
4. If conflict-free: print the planned actions.
5. Mutate only when the command's execution mode permits it (§2, §4): every
   mutating verb (`deploy`/`undeploy`/`redeploy`/`adopt`/`eject`) is dry-run by
   default and mutates only with `--apply`.
6. Apply actions in listed order. For `redeploy`, all removals precede all
   creations.

Collision and deduplication rules:

- Duplicate package names within one invocation are de-duplicated; the package
  is planned once.
- Two **different** packages producing the same leaf target is a
  `multiple_providers` conflict ([§12.6](#126-conflict-rules)).
- Two planned actions targeting the same path with incompatible types (e.g. one
  package plans `mkdir P` while another plans `symlink` at `P`) is a planned
  collision and is reported as a conflict; the plan does not apply.

### 8.1 Privilege (root context)

`tuck` never silently self-escalates. Privilege is decided by a **preflight
check**, before any mutation, and is **separate** from where the root context's
filesystem writes land. It is not inferred from whether the target root happens
to be writable: a root-context plan may touch read-only subtrees, and
`remove_symlink`/`move` depend on parent directories rather than the root
itself.

For `root`-context commands whose conflict-free plan contains **actions that
would write** under the root context (`mkdir`, `symlink`, `remove_symlink`,
`move`):

- The plan is **marked** as requiring privilege (`privilege.required`, §9.2.1).
  The marker is informational and tied to the context and action set; it does
  not by itself imply the command cannot apply.
- A separate **preflight predicate** determines whether the process is
  privileged to perform those mutations (`privilege.satisfied`). In production
  this is the process privilege (e.g. effective uid `0`, or the equivalent
  capability).
- When `--apply` is requested, `required` is true, and `satisfied` is false,
  `tuck` prints the plan, **performs no mutation**, and exits `5`, instructing
  the user to re-run under `sudo`. Because the check is preflight, exit `5`
  always leaves the target tree untouched; a filesystem error encountered *after*
  mutation has begun is exit `6` (§10), never `5`.
- A plan-only run (no `--apply`) only marks the requirement and exits `0`.
- A conflict-free plan with **no** write actions (a complete no-op), any
  plan-only run, and all read-only commands never require privilege.

---

## 9. Output formats

Every command supports two mutually exclusive modes: human (default) and
`--json`.

### 9.1 Human output

A plan renders as a header, a `plan:` block, an optional `conflicts:` block, and
a summary line.

```text
tuck deploy zsh   (context: home, dry-run)

plan:
  + mkdir  ~/.config/zsh
  + link   ~/.config/zsh/.zshrc -> ~/.dotfiles/zsh/.config/zsh/.zshrc

2 actions, 0 conflicts
```

Action glyphs: `+ mkdir`, `+ link` (symlink), `- unlink` (remove_symlink),
`~ move`. Conflicts use `!`. Paths under `$HOME` are abbreviated with `~`.

Conflict example:

```text
tuck deploy git   (context: home)

conflicts:
  ! ~/.gitconfig  target exists as real file
    hint: use `tuck adopt git ~/.gitconfig` to move it into a package

error: 1 conflict; nothing was changed
```

Resolved absolute paths and owning-package identities are available in full via
`--json` (§9.2).

### 9.2 JSON output

With `--json`, a command prints exactly one JSON document to stdout and nothing
else. Envelope:

```json
{
  "schemaVersion": 1,
  "command": "deploy",
  "context": "home",
  "kind": "plan",
  "data": { },
  "exitCode": 0
}
```

- `schemaVersion` — integer; incremented on breaking changes.
- `command` — the invoked command.
- `context` — `"home"` or `"root"`.
- `kind` — one of `plan`, `packages`, `tree`, `status`, `error`.
- `data` — payload determined by `kind` (below).
- `exitCode` — mirrors the process exit code.

#### 9.2.1 `kind: "plan"` (deploy / undeploy / redeploy / adopt / eject)

```json
{
  "schemaVersion": 1,
  "command": "deploy",
  "context": "home",
  "kind": "plan",
  "data": {
    "dryRun": true,
    "applied": false,
    "packages": ["public:home:zsh"],
    "privilege": { "required": false, "satisfied": true },
    "actions": [
      { "type": "mkdir", "path": "/home/me/.config/zsh" },
      {
        "type": "symlink",
        "linkPath": "/home/me/.config/zsh/.zshrc",
        "payload": "../../.dotfiles/zsh/.config/zsh/.zshrc",
        "target": "/home/me/.dotfiles/zsh/.config/zsh/.zshrc"
      }
    ],
    "conflicts": []
  },
  "exitCode": 0
}
```

A conflict object:

```json
{
  "code": "real_file",
  "targetPath": "/home/me/.gitconfig",
  "package": "public:home:git",
  "entry": "/home/me/.dotfiles/git/.gitconfig",
  "message": "target exists as real file",
  "hint": "use `tuck adopt git ~/.gitconfig` to move it into a package"
}
```

When conflicts are non-empty, `applied` is `false` and `exitCode` is `1`.

**Privilege-required (exit `5`).** `kind` stays `"plan"`; the plan is conflict-
free but not applied because `required` is true and `satisfied` is false:

```json
{
  "schemaVersion": 1, "command": "deploy", "context": "root",
  "kind": "plan",
  "data": {
    "dryRun": false,
    "applied": false,
    "packages": ["public:root:sshd"],
    "privilege": { "required": true, "satisfied": false, "reason": "root-context write requires elevated privileges" },
    "actions": [ { "type": "symlink", "linkPath": "/etc/ssh/sshd_config", "payload": "…", "target": "…" } ],
    "conflicts": []
  },
  "exitCode": 5
}
```

The `privilege` object reports the preflight policy (§8.1):

- `required` — boolean; the context is `root` and the plan contains write
  actions. Informational; present (as `false`) on every plan.
- `satisfied` — boolean; whether the preflight privilege predicate passed.
  Present whenever `required` is `true`.
- `reason` — string; included when `required` is `true`.

Exit `5` occurs exactly when `--apply` is given, `required` is `true`, and
`satisfied` is `false`. A root-context apply with `satisfied: true` proceeds
normally (`applied: true`, exit `0`).

**Runtime failure during apply (exit `6`).** `kind` stays `"plan"`; mutation
started but an action failed. `applied` is `false`, `completedActions` counts
actions that succeeded before the failure (not rolled back), and `failure`
identifies the failing action:

```json
{
  "schemaVersion": 1, "command": "adopt", "context": "home",
  "kind": "plan",
  "data": {
    "dryRun": false,
    "applied": false,
    "completedActions": 1,
    "actions": [ { "type": "mkdir", "path": "…" }, { "type": "move", "src": "…", "dst": "…" } ],
    "failure": { "actionIndex": 1, "code": "io_error", "message": "permission denied" },
    "conflicts": []
  },
  "exitCode": 6
}
```

For a successfully applied plan, `applied` is `true` and `completedActions`
equals the number of actions.

#### 9.2.2 `kind: "packages"`

```json
{
  "schemaVersion": 1, "command": "packages", "context": "home",
  "kind": "packages",
  "data": {
    "source": "public",
    "packages": ["git", "zsh"]
  },
  "exitCode": 0
}
```

#### 9.2.3 `kind: "tree"`

```json
{
  "schemaVersion": 1, "command": "tree", "context": "home",
  "kind": "tree",
  "data": {
    "packages": [
      {
        "identity": "public:home:zsh",
        "root": "/home/me/.dotfiles/zsh",
        "entries": [
          { "rel": ".config/zsh",        "type": "dir"  },
          { "rel": ".config/zsh/.zshrc", "type": "leaf" }
        ]
      }
    ]
  },
  "exitCode": 0
}
```

#### 9.2.4 `kind: "status"`

```json
{
  "schemaVersion": 1, "command": "status", "context": "home",
  "kind": "status",
  "data": {
    "entries": [
      {
        "targetPath": "/home/me/.config/zsh/.zshrc",
        "state": "deployed",
        "package": "public:home:zsh",
        "entry": "/home/me/.dotfiles/zsh/.config/zsh/.zshrc"
      },
      {
        "targetPath": "/home/me/.gitconfig",
        "state": "conflict",
        "code": "real_file",
        "message": "target exists as real file"
      }
    ]
  },
  "exitCode": 0
}
```

`state` is one of: `deployed`, `absent`, `conflict`, `mismatch`,
`owned_by_other`, `unmanaged`.

#### 9.2.5 `kind: "sources"`

Emitted by `source list` (and the data block written/echoed by `source enable`):

```json
{
  "schemaVersion": 1, "command": "source", "context": "home",
  "kind": "sources",
  "data": {
    "sources": [
      { "id": "public",  "path": "/home/me/.dotfiles",         "enabled": true, "default": true  },
      { "id": "private", "path": "/home/me/.dotfiles-private", "enabled": true, "default": false }
    ]
  },
  "exitCode": 0
}
```

#### 9.2.6 `kind: "error"`

```json
{
  "schemaVersion": 1, "command": "deploy", "context": "home",
  "kind": "error",
  "data": {
    "code": "package_not_found",
    "message": "package \"ssh\" not found in source \"public\"",
    "hint": "pass --source private if it lives there, or check `tuck packages`",
    "details": { "ref": "ssh", "source": "public", "context": "home" }
  },
  "exitCode": 4
}
```

---

## 10. Exit codes

| Code | Name | Meaning |
| --- | --- | --- |
| `0` | OK | Success: applied cleanly, dry-run printed, or read completed. |
| `1` | Conflict | Plan had conflicts; nothing was mutated. |
| `2` | Usage | Invalid flags/arguments/command, or invalid package-ref syntax. |
| `3` | Config/state | Machine state invalid, a repository manifest missing/invalid, an enabled source root missing or overlapping, or no source selected (`no_source`). |
| `4` | Resolution | Package not found in the active source, or an unknown/disabled `--source` id. |
| `5` | Privilege | Root-context mutation requires elevated privileges. |
| `6` | Runtime | A filesystem error occurred while applying a conflict-free plan (or writing machine state). |

Notes:

- `status` never returns `1` for in-body conflicts; it reports them in `data`
  and exits `0`.
- A partial failure during apply (exit `6`) stops at the failing action; already
  applied actions are not rolled back. Human output and the JSON `failure`
  object (with `actionIndex`/`completedActions`, §9.2.1) name the failing action.

### 10.1 Error message format (human)

```text
error: <message>
hint: <actionable suggestion>     # optional, when one applies
```

Messages are lowercase, specific, and reference the offending path or ref.
Standard error `code` values (stable identifiers used in JSON and tests)
include: `usage`, `manifest_missing`, `manifest_invalid`, `state_invalid`,
`source_root_missing`, `unknown_source`, `no_source`, `package_not_found`,
`invalid_ref`, `real_file`, `real_directory`, `special_file`,
`unmanaged_symlink`, `owned_by_other`, `path_mismatch`, `multiple_providers`,
`outside_target_root`, `inside_source_repo`, `package_path_exists`,
`not_a_managed_symlink`, `privilege_required`, `io_error`.

---

## 11. Help and usage text

`tuck --help` (top-level):

```text
tuck — manage dotfiles by linking package leaves into a target tree

usage:
  tuck [global-flags] <command> [args]

commands:
  deploy    <package>...           create managed links for a package
  undeploy  <package>...           remove managed links for a package
  redeploy  <package>...           refresh managed links (undeploy + deploy)
  adopt     <package> <file>       move a real file into a package, then link it
  eject     <link>                 remove a managed link, restoring the real file
  packages                         list the active source's packages
  tree      [package]              show package contents
  status    [package] [--path P]   show managed/conflict state
  source    enable <path>          enable a dotfiles repo on this machine
  source    list                   list enabled sources

global flags:
      --root                use the root context (target /); default is home
  -s, --source ID           select active source by enabled id
      --json                machine-readable output
      --apply               execute the plan (mutating verbs plan only without it)
      --no-color            disable colored output (implied by --json)
  -V, --version             print version
  -h, --help                show help

run `tuck <command> --help` for command-specific help.
```

Per-command help (e.g. `tuck adopt --help`) prints the command synopsis, its
arguments, the flags that affect it, its default execution mode, and one
worked example.

---

## 12. Resolution algorithms

This section is **normative** and self-contained: it specifies the internal
resolution, ownership, conflict, and operation algorithms that the command
reference (§7) and plan model (§8) build on. Pseudocode is illustrative; the
described behavior is authoritative. The `tuck` model is deliberately narrow:
package directories map onto real target directories, only **leaf** entries are
linked, directory folding is never performed, and the process working directory
never affects correctness except when resolving an explicitly relative input
path.

### 12.1 Contexts, bases, and identities

The two contexts (§3.1) are defined by two helpers:

| Context | `packageBase(source, context)` | `targetRoot(context)` |
| --- | --- | --- |
| `home` | `<source.path>` | `$HOME` |
| `root` | `<source.path>/.root` | `/` |

- A **package base** is the directory that holds packages for one source and
  context. A **package root** is one concrete package directory inside a base:
  `packageRoot(source, context, name) = join(packageBase(source, context), name)`.
- A **package identity** is `source-id + context + package-name`, written in
  display form as `source:context:name` (e.g. `public:home:zsh`,
  `public:root:sshd`). The `source-id` is the **effective id** from machine-local
  state ([§5.3](#53-machine-local-state)) — the manifest `name`. Identities are
  internal/display only; CLI
  refs are plain names (§6) and the context comes from `--root`, never from the
  ref.
- A **managed entry** is a target symlink whose payload resolves to a path inside
  the **active source's** package root and whose target location matches the
  package-relative path. Ownership is inferred from the payload
  ([§12.5](#125-ownership-resolution)); no manifest of deployed links is kept.

### 12.2 Path primitives

#### Expand input path

For any user-supplied path: (1) expand a leading `~`; (2) if relative, resolve
against the process current working directory; (3) clean lexical components
(`.`, redundant separators); (4) do **not** follow the final component when the
command must inspect the symlink itself (`eject`, `adopt`, `status --path`).

#### Canonicalize source roots

For each enabled source and its context root:
expand `~`, make absolute, clean, and resolve symlinks **in the root itself**. A
source root that does not exist is a state error (exit `3`).

#### Check containment

`inside(child, root)` is true only when `child` equals `root` or is a descendant
after both are absolute and clean. The test is **path-segment aware**:
`/home/me/.dotfiles-private` is **not** inside `/home/me/.dotfiles`.

#### Convert package path → target path

```text
rel = relativePath(packageRoot, packageEntryPath)
reject if rel is "." or starts with ".."
targetPath = clean(join(targetRoot, rel))
reject if not inside(targetPath, targetRoot)
return targetPath
```

For `root`, `targetRoot` is `/`, so every absolute path is technically inside it;
root commands must still reject paths inside **any enabled source repository** to
avoid adopting the dotfiles repo into itself.

#### Convert target path → package path

```text
absTarget = expand input path
reject if not inside(absTarget, targetRoot)
rel = relativePath(targetRoot, absTarget)
reject if rel is "." or starts with ".."
packagePath = clean(join(packageRoot, rel))
reject if not inside(packagePath, packageRoot)
return packagePath
```

### 12.3 Source and package resolution

#### Select active source

Every command that operates on a single source resolves it (cf. §3.2):

```text
if --source given:
    look up enabled source by id            (else unknown_source, exit 4)
else if a machine-local default source is set: use it
else if exactly one source is enabled:         use it
else:                                          error no_source (exit 3)
```

The path-based commands `eject` and `status --path` select the active source the
same way and infer ownership **within it** ([§12.5](#125-ownership-resolution));
`--source` is valid on them.

#### Parse package reference

A ref is a plain `<package-name>` (§6). Reject (usage error, exit `2`): any `:`
(source qualification is not supported — use `--source`), an empty name, an
absolute name, a name with `..` segments, or a name containing a path separator
(packages are direct children of the base; this keeps the ownership-inference
key — the first path segment — unambiguous).

#### Resolve existing package

For `deploy`, `undeploy`, `redeploy`, `tree <ref>`, and `status <ref>`:

```text
parse ref
source = select active source
packageRoot = packageRoot(source, context, name)
if packageRoot does not exist: error package_not_found (exit 4)
return source, context, name, packageRoot
```

There is **no** cross-source search and therefore no ambiguity case: a name
either exists in the active source/context or it does not.

#### Resolve package for adopt

`adopt` may create a new package path, but the source is still fixed:

```text
parse ref
source = select active source
packageRoot = packageRoot(source, context, name)   # may or may not exist yet
return source, context, name, packageRoot
```

Because the source is fixed there is no "new package requires source
qualification" case.

### 12.4 Package entry enumeration

```text
walk packageRoot depth-first
for each entry (skip packageRoot itself):
    rel = relativePath(packageRoot, entry)
    reject if rel escapes packageRoot
    if entry is a directory: record a directory entry
    else:                    record a leaf entry
```

Rules: directory entries cause real directories to be created in the target tree
and are **never** represented as symlinks; leaf entries are linked. If a package
leaf is itself a symlink, the target link points at the package symlink **itself**
(not its resolved destination), keeping ownership local to the package tree.

### 12.5 Ownership resolution

#### Classify target path

Input: `targetPath`, optional selected package identity, context, the **active
source**.

```text
stat = lstat(targetPath)
if not exists:                         return Absent
if directory and not symlink:          return RealDirectory
if regular file and not symlink:       return RealFile
if other non-symlink type:             return SpecialFile
# symlink:
owner = inferSymlinkOwner(targetPath, context, active source)
if owner is ManagedPathMismatch:       return ManagedPathMismatch(owner)
if owner is none:                      return UnmanagedSymlink
if no selected package:                return ManagedSymlink(owner)
if owner == selected package:          return ManagedBySelectedPackage(owner)
return ManagedByOtherPackage(owner)
```

#### Infer symlink owner

Ownership inference scans only the **active source** ([§3.2](#32-source-selection)).
A symlink that points into a different (non-active) source therefore reads as
`none`/`UnmanagedSymlink`; select that source with `--source` to operate on it.
Because enabled source roots may not overlap ([§5.4](#54-validation)), the single
active base cannot misattribute a link that belongs to another enabled repo:

```text
payload = readlink(targetLinkPath)
targetAbs = payload is relative
              ? clean(join(dirname(targetLinkPath), payload))
              : clean(payload)

base = packageBase(activeSource, context)
if not inside(targetAbs, base): return none
relToBase   = relativePath(base, targetAbs)
packageName = first path segment of relToBase
packageRoot = join(base, packageName)
packageRel  = relativePath(packageRoot, targetAbs)
expectedTarget = clean(join(targetRoot(context), packageRel))
if clean(targetLinkPath) != expectedTarget:
    return ManagedPathMismatch(activeSource, context, packageName, packageRel, expectedTarget)
return ManagedOwner(activeSource, context, packageName, packageRoot, packageRel)
```

Notes: ownership needs no manifest; broken symlinks are still classifiable if
their lexical target is inside a package root; a managed symlink whose link path
does not match its package-relative path is reported as a mismatch and is never
mutated automatically.

### 12.6 Conflict rules

**Deploy (leaf).** A leaf target is **linkable** when it is absent, or already a
symlink owned by the selected package pointing at the same entry. It **conflicts**
when it is a real file, a real directory, a special file, an unmanaged symlink,
managed by another package, or managed by the selected package but mapping to a
different package-relative path.

**Directory.** A package directory entry's target is valid when absent (created
as a real directory) or already a real directory. It conflicts when the target is
a file, a symlink, or any non-directory special file.

**Adopt.** Requires: the target exists, is a real file (not a symlink, not a
directory), is inside the selected target root, is **not** inside any enabled
source repository, and the destination package path
does not already exist.

**Eject.** Requires: the target is a symlink managed in the selected context, the
managed package file exists and is **not** a directory, the symlink path matches
the package-relative target path, and materializing the real file does not
overwrite unrelated content.

### 12.7 Operation algorithms

Every mutating command first builds a **complete** plan; if any conflict is
found it prints the conflicts and mutates nothing (§8). All five are dry-run by
default and mutate only with `--apply` (§2, §4).

#### `deploy`

```text
resolvedPackages = resolve existing package for each ref
plannedTargets = {}            # targetPath -> packageEntry

for each package:
    for each directory entry:
        targetDir = convert package path -> target path
        switch classify(targetDir):
            Absent:        plan mkdir targetDir
            RealDirectory: no-op
            else:          conflict
    for each leaf entry:
        targetPath = convert package path -> target path
        if targetPath in plannedTargets with a different entry:
            conflict multiple_providers; continue
        switch classify(targetPath, selected package):
            Absent:                              plan symlink targetPath -> entry
            ManagedBySelectedPackage(same entry): no-op
            else:                                conflict
```

Symlink payloads are relative:
`payload = relativePath(dirname(targetPath), packageEntryPath)`.

#### `undeploy`

```text
resolvedPackages = resolve existing package for each ref
for each package, for each leaf entry:
    targetPath = convert package path -> target path
    switch classify(targetPath, selected package):
        Absent:                              no-op
        ManagedBySelectedPackage(same entry): plan remove_symlink targetPath
        else:                                conflict / "not owned by selected package"
```

Directories are **never** pruned (the tool does not record which real
directories it created); a future `prune`/`doctor` command could remove empty
directories under explicit intent.

#### `redeploy`

```text
build undeploy plan for the selected packages; if conflicts: stop
build deploy plan against the post-undeploy state; if conflicts: stop
apply all remove_symlink actions, then all symlink/mkdir actions
```

An already-owned link may be removed and recreated so its payload is normalized
to the preferred relative form.

#### `adopt`

```text
targetPath = expand input path (do not follow final symlink)
reject if not inside(targetPath, targetRoot(context))
reject if inside(targetPath, any enabled source repository)
reject unless classify(targetPath) is RealFile

(source, context, name, packageRoot) = resolve package for adopt
packagePath = convert target path -> package path
reject if packagePath already exists
reject if not inside(packagePath, packageRoot)

plan mkdir dirname(packagePath)
plan move targetPath -> packagePath
plan symlink targetPath -> packagePath        # payload relative, as in deploy
```

#### `eject`

```text
targetPath = expand input path (do not follow final symlink)
reject unless classify(targetPath, context) is ManagedSymlink
owner = classification.owner

packagePath    = join(owner.packageRoot, owner.packageRel)
expectedTarget = join(targetRoot(context), owner.packageRel)
reject if targetPath != expectedTarget
reject if packagePath does not exist
reject if packagePath is a directory

plan remove_symlink targetPath
plan move packagePath -> targetPath
```

The now-empty package directory is left in place.

### 12.8 Listing algorithms

#### `packages`

```text
base = packageBase(activeSource, context)
list direct child directories of base as packages
    skip the reserved `.root` directory (it is the root-context base, not a package)
    (tuck.toml is a file at the source root, not a directory, so it is never a package)
```

#### `tree`

Without a ref: show the **active source's** package tree grouped by package. With
a ref: resolve it in the active source
([§12.3](#123-source-and-package-resolution)) and show that package root's tree.

### 12.9 Execution planning

Mutations are the explicit actions defined in §8 (`mkdir`, `symlink`,
`remove_symlink`, `move`). Planning rules: resolve the active source, packages,
target paths, and ownership before mutating; accumulate **all** conflicts (do not
stop at the first); on any conflict print them and exit `1` without mutating; on a
conflict-free plan print the actions and mutate only when `--apply` is given
(§8). Root-context mutations make their privilege requirement visible in the plan
and never self-escalate (§8.1).

---

## Appendix A — Worked examples

### A.0 Bootstrap a new machine

```text
$ git clone git@github.com:me/dotfiles.git ~/.dotfiles   # repo contains tuck.toml (name = "public")
$ tuck source enable ~/.dotfiles
enabled source "public" -> /home/me/.dotfiles (default)

$ tuck source list
* public   /home/me/.dotfiles          (default)

$ tuck deploy zsh --apply
2 actions, 0 conflicts — applied
```

`* ` marks the machine-local default source. Before the first `source enable`, any
command needing an active source fails: `error: no source selected; run
\`tuck source enable <path>\` or pass --source` (exit 3).

### A.1 Home deploy

```text
$ tuck deploy zsh
tuck deploy zsh   (context: home, dry-run)

plan:
  + mkdir  ~/.config/zsh
  + link   ~/.config/zsh/.zshrc -> ~/.dotfiles/zsh/.config/zsh/.zshrc

2 actions, 0 conflicts
re-run with --apply to execute

$ tuck deploy zsh --apply
2 actions, 0 conflicts — applied
```

### A.2 Package not found in the active source

```text
$ tuck deploy ssh                 # active source = public (the default)
error: package "ssh" not found in source "public"
hint: pass --source private if it lives there, or check `tuck packages`
# exit 4
```

### A.3 Selecting a different source

```text
$ tuck --source private deploy ssh --apply   # source=private, context=home, root=~/.dotfiles-private/ssh
```

### A.4 Root package

```text
$ tuck --root deploy sshd        # context=root, root=~/.dotfiles/.root/sshd, target=/
# dry-run by default; add --apply to execute
# package file ~/.dotfiles/.root/sshd/etc/ssh/sshd_config -> /etc/ssh/sshd_config
# if unprivileged: `tuck --root deploy sshd --apply` prints plan, exits 5 (privilege required)
```

### A.5 Adopt an existing user file

```text
$ tuck --source private adopt ssh ~/.ssh/config   # dry-run by default
plan:
  + mkdir  ~/.dotfiles-private/ssh/.ssh
  ~ move   ~/.ssh/config -> ~/.dotfiles-private/ssh/.ssh/config
  + link   ~/.ssh/config -> ~/.dotfiles-private/ssh/.ssh/config

re-run with --apply to execute

$ tuck --source private adopt ssh ~/.ssh/config --apply   # applies
```

### A.6 Eject a managed file

```text
$ tuck --source private eject ~/.ssh/config --apply   # owner is in the private source
plan:
  - unlink ~/.ssh/config
  ~ move   ~/.dotfiles-private/ssh/.ssh/config -> ~/.ssh/config
applied
```

Without `--source private` (and with `public` as the default), the same link
points into a non-active source and reads as `unmanaged`:
`error: ~/.ssh/config is not a managed symlink in source "public"`.

### A.7 Deploy conflict

```text
$ tuck deploy git
conflicts:
  ! ~/.gitconfig  target exists as real file
    hint: use `tuck adopt git ~/.gitconfig` to move it into a package
error: 1 conflict; nothing was changed
# exit 1
```

### A.8 Managed by another package in the same source

```text
$ tuck deploy git                 # active source = public; ~/.gitconfig owned by public:home:work
conflicts:
  ! ~/.gitconfig  already managed by public:home:work
error: 1 conflict; nothing was changed
# exit 1
```

---

## Appendix B — Relationship to the draft

This spec **redesigns the CLI surface from first principles**; the CLI sketch in
the earlier `resolution-algorithms.md` draft (since removed) was
non-authoritative input only. Deliberate overrides:

1. **Command names.** `link/unlink/relink/capture/release` →
   `deploy/undeploy/redeploy/adopt/eject`. The two verb sets are now
   distinguished by whether a real file moves, and `eject` (not `undeploy`) is
   the materialization workflow — removing the draft's documented naming
   footgun.
2. **Context naming.** The draft's default context `user` is renamed to `home`,
   so both contexts are named after their target root (`home` → `$HOME`,
   `root` → `/`) rather than mixing a privilege-noun with a location-noun.
   Display identities change accordingly (`public:user:zsh` → `public:home:zsh`).
   The `root` context keeps its name (and its `.root` package-base convention).
3. **Context selection.** The draft's `tuck root <command>` prefix is replaced
   by a single global boolean `--root` flag; `home` is the unconditional default
   context and `--root` selects the `root` context. This avoids a parser
   special-case (a "root subcommand" vs. a package literally named `root`) and
   applies uniformly to every command.
   - *Rejected alternative:* a `--context home|root` flag. Dropped because there
     are exactly two contexts and `home` is always the default, so a boolean
     `--root` is simpler and there is nothing to set a default for.
   - *Rejected alternative:* keep `tuck root …` as a sugar alias. Dropped to
     keep a single, unambiguous invocation grammar.
   - *Rejected alternative:* user-**configurable** contexts (arbitrary
     `[contexts.*]` beyond home/root). Deferred (YAGNI): `$HOME` and `/` cover
     known dotfiles use cases, and the internals already treat context as
     pluggable (`packageBase(source, context)` / `targetRoot(context)`), so a
     third context can be added later without redesign.
4. **Source selection & configuration.** The draft's `[source:]name` ref grammar
   and cross-source ambiguity scan are removed, and the central config file is
   removed entirely. Every command operates on exactly one **active source**,
   chosen by `--source <id>` > the machine-local default > the sole enabled
   source, else `no_source` (exit `3`). Sources are no longer declared in a
   central config; each repo carries a committed `tuck.toml` manifest (its
   `name`), and machine-local state (`sources.toml`, written by `tuck source
   enable`) records which repos are enabled on a machine and where they live.
   This split keeps portable identity in the repo and machine-specific paths
   local, and dissolves the bootstrap loop of a repo-managed central config.
   Package refs are plain names; a missing package is `package_not_found` in the
   active source (no cross-source search).
   - *Consequence:* `adopt` no longer needs a "new package requires source
     qualification" rule (the source is fixed). Existing-package resolution is a
     direct lookup in the active source and context.
   - *No simultaneous multi-source:* `packages`/`tree` list only the active
     source, and `eject`/`status --path` infer ownership in the active source
     only (`--source` is valid on them). A link into a non-active source reads as
     `unmanaged` until that source is selected. This trades cross-source
     operations for deterministic, explicit operator control.
   - *Rejected alternative:* a central `config.toml` registry with
     `[sources.<id>]` + `[defaults].source`. Dropped because source paths are
     inherently machine-specific (so a committed registry is not portable) and a
     repo-managed central config creates a chicken-and-egg loop on a new machine.
   - *Rejected alternative:* keep the inline `source:name` qualifier (a hybrid).
     Dropped in favor of a single, unambiguous selection mechanism.
5. **Machine output.** A stable, versioned `--json` mode is mandatory for every
   command; the draft did not specify one.
6. **Exit codes & error codes.** A fixed taxonomy (§10) is defined; the draft
   only showed sample messages.
7. **Privilege.** Root-context mutations surface required privilege and exit `5`
   rather than self-escalating; the draft left escalation unspecified.
8. **Confirmation model.** All five mutating verbs
   (`deploy`/`undeploy`/`redeploy`/`adopt`/`eject`) **plan by default and mutate
   only with `--apply`**; the draft auto-applied symlink operations.
   - *Rationale:* one uniform rule (no "which verbs auto-apply?" special case)
     and a safe default for writes to `$HOME` and `/`. The common workflow here
     is incremental `adopt`/`eject`; `deploy`/`undeploy` are rare bootstrap
     operations.
   - *Rejected alternative:* a `--dry-run`/`-n` flag. Dropped because plan-only
     is already the default, so the flag would be a pure no-op. It may return if
     the default apply/plan behavior is ever made configurable (e.g. a future
     machine-local preference).

The **internal algorithms** (path primitives, package-entry enumeration,
ownership inference, conflict rules, operation algorithms) have been **merged
into this document** as the normative [§12](#12-resolution-algorithms),
translated into the current vocabulary (`deploy`/`undeploy`/`redeploy`/`adopt`/
`eject`, `home`/`root`, single active source, `--apply`). The draft's logic is
preserved except for source/package-ref **resolution**, which is replaced by the
single-active-source model ([§3.2](#32-source-selection),
[§6](#6-package-references), [§12.3](#123-source-and-package-resolution)): no
cross-source search and no "ambiguous package" case. The original
`resolution-algorithms.md` draft has been removed now that its content lives
here.
