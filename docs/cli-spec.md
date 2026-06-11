# `tuck` CLI -- Implementation Specification

Status: authoritative for the command-line interface.
Scope: command surface, flags, configuration, output (human + JSON), error
reporting, plan rendering, and internal resolution algorithms.

This document is the source of truth for the CLI and its internal resolution
algorithms. It defines command semantics, source selection, path resolution,
planning, output envelopes, and error classification.

`tuck` is a Go dotfiles manager. It maps package directories onto real target
directories, links only leaf entries, never folds directories, and never lets the
caller's working directory affect correctness except when resolving an explicitly
relative input path.

---

## 1. Concepts

- **Source** -- a dotfiles repository. Each repository carries a committed
  manifest (`<repo>/tuck.toml`) declaring its short `name`, used as the source
  id. A source is made usable on a machine with `tuck source add <path>`, which
  records its path and enabled state in machine-local state.
- **Target context** -- selects where package entries appear:
  - `home`: package base `<source.path>`, target root `$HOME` (default).
  - `root`: package base `<source.path>/.root`, target root `/`.
- **Package** -- one directory inside a package base. It is identified
  internally as `source:context:name` (for example `public:home:zsh`). CLI input
  never includes source or context; source is selected with `--source` or
  machine state, and context is selected with `--root`.
- **Package reference** -- CLI input naming a package by its plain `<name>` (for
  example `zsh`). Package refs do not contain source, context, path separators,
  or `:`.
- **Managed entry** -- a target symlink whose resolved destination is inside the
  active source's package root at the matching package-relative path. Ownership
  is inferred from the link payload; there is no deployed-link manifest.
- **Leaf vs. directory entries** -- only leaf entries become symlinks; directory
  entries become real directories in the target tree.

---

## 2. Command surface

Command depth mirrors operation frequency:

- file operations are top-level commands;
- package operations live under `package` (alias `pkg`);
- source operations live under `source`.

```text
tuck [global-flags] <command> [command-args] [command-flags]
```

Synopsis:

```text
# File operations (top-level)
tuck adopt  <file> <package-ref>
tuck eject  <file>
tuck status <file>

# Package operations
tuck package use     <package-ref>...
tuck package use     --all
tuck package drop    <package-ref>...
tuck package refresh <package-ref>...
tuck package list
tuck package show    <package-ref>
tuck package status  [package-ref]

# Source operations
tuck source add     <path> [--default]
tuck source rm      <id>
tuck source list
tuck source default <id>
```

Aliases:

| Canonical | Alias | Scope |
| --- | --- | --- |
| `package` | `pkg` | command group |
| `package list` | `package ls`, `pkg ls` | package group |
| `package show` | `package tree`, `pkg tree` | package group |
| `source list` | `source ls` | source group |

Semantics:

- `adopt` -- move one existing real target file into a package, then create a
  managed symlink pointing back to it.
- `eject` -- remove one managed target symlink and move the package file back to
  the target tree.
- `status <file>` -- classify a single target path in the active source/context.
- `package use` -- create managed target symlinks for package leaf entries.
- `package drop` -- remove managed target symlinks for selected packages. It
  does not move package files back into the target tree.
- `package refresh` -- rebuild selected package links (drop then use), also
  normalizing symlink payloads to the preferred relative form.
- `package list` -- list packages in the active source and context.
- `package show` -- display one package's contents.
- `package status` -- report managed/conflicting/absent state for package
  entries; without a package, summarize all packages in the active source.
- `source add` -- register and enable a dotfiles repository on this machine.
- `source rm` -- remove a source from machine-local state.
- `source list` -- list sources recorded on this machine.
- `source default` -- set the machine-local default active source.

All mutating commands plan by default and mutate only with `--apply`.

| Set | Commands | Moves real files? | Default execution |
| --- | --- | --- | --- |
| File movement | `adopt`, `eject` | Yes | dry-run; needs `--apply` |
| Package link management | `package use`, `package drop`, `package refresh` | No | dry-run; needs `--apply` |
| Read-only | `status`, `package list`, `package show`, `package status`, `source list` | No | n/a |
| Source management | `source add`, `source rm`, `source default` | No target-tree writes | immediate machine-state write |

---

## 3. Invocation model

- Unknown commands, missing help topics, unknown flags, and related CLI shell
  errors follow urfave/cli's framework behavior. They exit `1`.
- With no command, `tuck` prints top-level help and exits `0`. A future version
  may use bare `tuck` as a TUI entry point.
- Invoking a command group with no subcommand (`tuck package`, `tuck source`)
  prints that group's help and exits `0`.
- `--help`, `--version`, the no-command case, and command-group help bypass
  machine-state discovery and source resolution.
- `--help --json` and `--version --json` emit machine-readable metadata for
  tooling. Plain help/version remains the default. There is no `help` command;
  help is exposed only through `-h`/`--help`.
- `tuck source add <path>` does not require an existing active source; it
  establishes one. `tuck source list` requires only readable machine state.

### 3.1 Context selection

There are exactly two target contexts. `home` is the unconditional default; the
boolean `--root` flag selects `root`:

- no `--root` -> `home` (package base `<source.path>`, target root `$HOME`);
- `--root` -> `root` (package base `<source.path>/.root`, target root `/`).

`--root` is scoped to commands that operate on target paths or packages:
`adopt`, `eject`, `status`, and all `package` subcommands. It is not valid on
`source` commands.

### 3.2 Source selection

Every command that operates on packages or target paths uses exactly one active
source. The active source is resolved in this order:

1. `--source <id>` (`-s`) on the command line.
2. The machine-local default source (`tuck source default <id>` or
   `tuck source add <path> --default`).
3. The sole enabled source, when exactly one source is enabled.
4. Otherwise, error `no_source`.

`--source` accepts an enabled source id only. It is scoped to `adopt`, `eject`,
`status`, and all `package` subcommands. It is not valid on `source` commands.

There is no simultaneous multi-source operation. The listing commands operate
only on the active source. `eject` and `status <file>` infer ownership in the
active source only. A managed symlink that points into a non-active source reads
as unmanaged unless that source is made active.

---

## 4. Flags

No flag is global unless it is meaningful for every command.

| Flag | Alias | Scope | Meaning |
| --- | --- | --- | --- |
| `--json` | | universal | Emit one JSON document instead of human output. |
| `--no-color` | | universal | Disable colored output. Kept for now; may later defer to `NO_COLOR` only. |
| `--help` | `-h` | universal | Print help for the program or command. |
| `--version` | `-v` | root only | Print version. |
| `--source <id>` | `-s` | domain commands | Select the active source by enabled id. |
| `--root` | | domain commands | Select the root context. |
| `--apply` | | mutating target-tree commands | Execute the plan. Without it, print the plan only. |
| `--default` | | `source add` | Make the added source the machine-local default. |
| `--all` | | `package use` | Use every package in the active source/context. |

Domain commands are `adopt`, `eject`, `status`, and all `package` subcommands.
Mutating target-tree commands are `adopt`, `eject`, `package use`,
`package drop`, and `package refresh`.

Flag interaction rules:

- Mutating target-tree commands build and print a plan unless `--apply` is
  given.
- `--json` implies `--no-color`.
- `package use` requires either one or more package refs or `--all`, but not
  both.

---

## 5. Configuration

`tuck` has no central config file. Configuration is split between two artifacts:

- a repository manifest committed in each dotfiles repo (`<repo>/tuck.toml`),
  which carries the repo's portable identity;
- machine-local state (`sources.toml`), generated by `tuck source` commands,
  which records which repos are enabled on this machine and where they live.

There is no `--config` flag and no `$TUCK_CONFIG`. There is no config setting for
the default context; `home` is always the default unless `--root` is passed.

### 5.1 Bootstrap

```text
git clone <repo> ~/.dotfiles
tuck source add ~/.dotfiles --default
tuck pkg use zsh git --apply
```

`tuck.toml` is a file at the repository root, not a package directory, so it is
ignored by package enumeration.

### 5.2 Repository manifest

Committed in the repo at `<repo>/tuck.toml`:

```toml
name        = "public"
description = "public dotfiles"

[package.zsh]

[[package.zsh.file]]
path = ".config/symlink-hostile-app/config"
deploy = "copy"
mode = "0600"
```

Fields:

- `name` (required) -- the repo's short id. Used as the source id and in display
  identities. Must not be empty and must not contain a path separator or `:`.
- `description` (optional) -- a human-readable label shown in `source list`.

Package/file metadata is optional and planned for First Release behavior:

- `[package.<name>]` declares policy for a package whose package directory is
  still `<source>/<name>` (or `<source>/.root/<name>` in root context).
- `[[package.<name>.file]]` declares policy for one package-relative leaf path.
- `path` (required) is the package-relative file path, using the same path that
  maps to the target tree.
- `deploy` (optional, default `"symlink"`) selects the deployment strategy for
  that leaf. First Release values are `"symlink"` and `"copy"`. Hardlinks are a
  possible future strategy but are not committed behavior in this spec.
- `mode` (optional) is an explicit octal file mode such as `"0600"`. It applies
  after creating a copied target and to other target entries where mode is
  meaningful. Owner/group management is out of scope.

Package directories remain target-tree mirrors. Portable policy belongs in the
repo manifest so mode and deployment overrides are visible in one audited file,
not scattered through package-local metadata files.

The format is open to additive keys. Unknown top-level keys are ignored so that
newer repos remain readable by older binaries. A missing or unreadable
`tuck.toml` is `manifest_missing`; a malformed manifest or invalid/missing
`name` is `manifest_invalid`.

### 5.3 Machine-local state

Generated and updated by `tuck source` commands; not committed. Location:

```text
${XDG_STATE_HOME:-~/.local/state}/tuck/sources.toml
```

For tests, the state directory may be overridden by `TUCK_TEST_STATE_DIR`, which
is compiled in only under the `tuck_testhooks` build tag.

```toml
default = "public"
checksum = "sha256:..."

[[source]]
path    = "/home/me/.dotfiles"
id      = "public"
enabled = true

[[source]]
path    = "/home/me/.dotfiles-private"
id      = "private"
enabled = true

[[copy]]
source  = "public"
context = "home"
package = "zsh"
path    = ".config/symlink-hostile-app/config"
target  = "/home/me/.config/symlink-hostile-app/config"
sourceChecksum = "sha256:..."
targetChecksum = "sha256:..."
```

Fields per `[[source]]` entry:

- `path` (required) -- the canonical repository path on this machine.
- `id` (required) -- the effective source id, authoritative for selection and
  display identities. In MVP it is always the manifest `name`.
- `enabled` (optional, default `true`) -- whether the source participates.

Top-level fields:

- `default` (optional) -- the effective id of the machine-local default active
  source. Default status belongs only to the registry, never to individual source
  entries.
- `checksum` (optional, generated) -- a checksum over the normalized state file
  or over a generated sidecar payload. It is a fast validation signal, not a
  security boundary.

First Release copied-file state:

- `[[copy]]` entries record copied targets because copy ownership cannot be
  inferred from symlink payloads.
- `source`, `context`, `package`, and `path` identify the package leaf.
- `target` records the target path for status/drop operations.
- `sourceChecksum` and `targetChecksum` record the last applied source and target
  bytes so status can distinguish unchanged copies, source drift, target drift,
  and both-sides drift.

If the state file is absent or has no entries, no source is enabled. A command
that needs an active source fails with `no_source`. Reading the state for
`source list` is not itself an error.

The state file remains human-readable text. `tuck` may also maintain a generated
checksum sidecar (for example `sources.toml.sha256`) or equivalent checksum field
to detect accidental manual edits/truncation quickly. A mismatch is reported as
`state_checksum_mismatch` with a repair hint; it is not a tamper-proof security
mechanism.

### 5.4 Validation

Performed when state is loaded:

- Effective `id` values are unique across enabled entries and must not be empty
  or contain a path separator or `:`.
- If top-level `default` is set, it must name an enabled entry.
- Each enabled `path` is expanded and canonicalized. A missing path is a state
  error.
- Enabled source roots must not overlap: no enabled `path` may equal, contain,
  or be contained by another enabled `path` (path-segment aware).
- Each enabled `path` must contain a readable, valid `tuck.toml`.
- For the `root` context, the package base is `<source.path>/.root`. A source
  that has no `.root` directory contributes no packages in the root context but
  is not itself an error.

State and manifest failures exit `1`; `--json` exposes the stable `error.code`.

---

## 6. Package references

A package reference is a plain package name:

```text
package-ref := package-name
```

Examples: `zsh`, `ssh`, `git`.

Rejected before package resolution:

- a ref containing `:`;
- an empty package name;
- a package name starting with `.`;
- an absolute package name;
- a name containing `..` path segments;
- a name containing a path separator.

Resolution rules:

- `package use`, `package drop`, `package refresh`, `package show`, and
  `package status <ref>` require the package to exist in the active source and
  context, else `package_not_found`.
- `adopt <file> <ref>` may create the package if absent. The source is still
  fixed by active-source resolution.

---

## 7. Command reference

Every mutating target-tree command first builds a complete plan. If any conflict
is found, it prints conflicts and mutates nothing.

### 7.1 `adopt`

```text
tuck [--json] adopt [--source <id>] [--root] [--apply] <file> <package-ref>
```

- **Arguments:** one existing real target file and one package name. The package
  may already exist or be created in the active source.
- **Behavior:** expand `<file>` without following the final symlink; reject if
  outside the target root or inside any enabled source repository; reject unless
  it is a real file. Convert the target path to the destination package path and
  reject if that path already exists. Plan: make package parents, move
  target-to-package, symlink target-to-package.
- **Execution:** dry-run by default; mutates only with `--apply`.

### 7.2 `eject`

```text
tuck [--json] eject [--source <id>] [--root] [--apply] <file>
```

- **Arguments:** one target path that is a managed symlink.
- **Behavior:** expand `<file>` without following the final symlink; classify in
  the active source/context; reject unless it is a managed symlink whose link
  path matches the package-relative target and whose package file exists and is
  not a directory. Plan: remove symlink, move package-to-target, and remove
  now-empty intermediate source/package directories below the package root.
- **Execution:** dry-run by default; mutates only with `--apply`.

### 7.3 `status`

```text
tuck [--json] status [--source <id>] [--root] <file>
```

- **Arguments:** one target path.
- **Behavior:** classify the target path as absent, managed, unmanaged,
  mismatched, owned by another package, real file, real directory, or special
  file. Ownership is inferred only in the active source.
- **Execution:** read-only.

### 7.4 `package use`

```text
tuck [--json] package use [--source <id>] [--root] [--apply] <package-ref>...
tuck [--json] package use [--source <id>] [--root] [--apply] --all
```

- **Arguments:** one or more existing package names, or `--all`.
- **Behavior:** enumerate package entries; plan `mkdir` for absent directory
  entries and `symlink` for linkable leaf entries; treat conflicts as conflicts.
  Symlink payloads are relative.
- **Execution:** dry-run by default; mutates only with `--apply`.

### 7.5 `package drop`

```text
tuck [--json] package drop [--source <id>] [--root] [--apply] <package-ref>...
```

- **Arguments:** one or more existing package names.
- **Behavior:** for each leaf entry, plan `remove_symlink` only when the target is
  a symlink managed by the selected package for the same entry. Absent targets
  are no-ops; anything else is a conflict. Directories are never pruned.
- **Execution:** dry-run by default; mutates only with `--apply`.

### 7.6 `package refresh`

```text
tuck [--json] package refresh [--source <id>] [--root] [--apply] <package-ref>...
```

- **Arguments:** one or more existing package names.
- **Behavior:** build a drop plan, then a use plan against the post-drop state.
  Already-owned links may be removed and recreated to normalize relative
  payloads. Apply removals before creations.
- **Execution:** dry-run by default; mutates only with `--apply`.

### 7.7 `package list`

```text
tuck [--json] package list [--source <id>] [--root]
```

- **Behavior:** list direct child package directories of the active source's
  package base. Skip dot-prefixed directories (including `.root`) and
  non-directories such as `tuck.toml`.
- **Aliases:** `package ls`, `pkg list`, `pkg ls`.

### 7.8 `package show`

```text
tuck [--json] package show [--source <id>] [--root] <package-ref>
```

- **Behavior:** resolve the package in the active source/context and show its
  file tree.
- **Aliases:** `package tree`, `pkg show`, `pkg tree`.

### 7.9 `package status`

```text
tuck [--json] package status [--source <id>] [--root] [package-ref]
```

- **With `<package-ref>`:** resolve the package and report each leaf entry as
  `deployed`, `absent`, `conflict`, `mismatch`, or `owned_by_other`.
- **Without a ref:** summarize every package in the active source/context.
- **Execution:** read-only. Reported conflicts in the body do not make the
  command fail; it exits `0` when the query succeeds.

### 7.10 `source add`

```text
tuck [--json] source add <path> [--default]
```

- **Arguments:** one repository path.
- **Behavior:** read `<path>/tuck.toml`; record or update the source entry in
  machine-local state with canonical path, manifest id, `enabled = true`, and
  top-level default per `--default`. Validate the complete write; on validation
  failure, write nothing.

### 7.11 `source rm`

```text
tuck [--json] source rm <id>
```

- **Arguments:** one source id.
- **Behavior:** remove the entry from machine-local state. If it was the default,
  clear the default. Removing a missing source is an error (`unknown_source`).

### 7.12 `source list`

```text
tuck [--json] source list
```

- **Behavior:** list source entries from machine-local state: id, path, enabled,
  and default marker. Absent or empty state reports "no sources enabled" and exits
  `0`.
- **Aliases:** `source ls`.

### 7.13 `source default`

```text
tuck [--json] source default <id>
```

- **Arguments:** one enabled source id.
- **Behavior:** set the top-level machine-local default source id. Unknown or
  disabled source ids are errors.

---

## 8. Plan and action model

Mutating commands that plan filesystem changes emit an ordered list of actions:

| Action | Fields | Meaning |
| --- | --- | --- |
| `mkdir` | `path` | Create a real directory in the target tree. |
| `rmdir` | `path` | Remove an empty directory left behind in a source package tree. |
| `symlink` | `linkPath`, `payload`, `target` | Create a symlink. `payload` is relative link text; `target` is the resolved destination. |
| `copy` | `src`, `dst`, `mode` | Copy a package file to the target tree and optionally set its mode. |
| `remove_symlink` | `path` | Remove a managed target symlink. |
| `remove_copy` | `path` | Remove a tracked copied target. |
| `move` | `src`, `dst` | Move a real file. |

Planning rules:

1. Resolve all sources, packages, target paths, and ownership before mutating.
2. Accumulate all conflicts; do not stop at the first.
3. If any conflict exists, print conflicts, exit `1`, and mutate nothing.
4. If conflict-free, print the planned actions.
5. Mutate only when `--apply` is given.
6. Apply actions in listed order. For `package refresh`, removals precede
   creations.

Collision and deduplication rules:

- Duplicate package names within one invocation are de-duplicated.
- Two different packages producing the same leaf target are a
  `multiple_providers` conflict.
- Two planned actions targeting the same path with incompatible types are a
  conflict.
- A copied target is mutable outside `tuck`. If the target has drifted from the
  last recorded `targetChecksum`, `package use`, `package drop`, and
  `package refresh` must report a drift conflict rather than overwrite or remove
  it silently.

### 8.1 Privilege (root context)

`tuck` never silently self-escalates. Privilege is decided by a preflight check
before any mutation and is separate from where test writes land.

For root-context commands whose conflict-free plan contains write actions:

- The plan is marked as requiring privilege (`privilege.required`).
- A separate predicate determines whether privilege is satisfied.
- When `--apply` is requested, privilege is required, and privilege is not
  satisfied, `tuck` prints the plan, performs no mutation, exits `1`, and reports
  `error.code = "privilege_required"`.
- Plan-only runs mark the requirement and exit `0`.
- Read-only commands never require privilege.

---

## 9. Output formats

Every command supports human output (default) and `--json`.

**Streams.** Primary results -- plans, listings, status, and JSON envelopes -- go
to stdout. Diagnostics -- error messages and hints -- go to stderr. With
`--json`, exactly one JSON document is written to stdout and stderr stays empty.

### 9.1 Human output

A plan renders as a header, a `plan:` block, optional `conflicts:` block, and
summary. Exact glyphs and color are not locked down yet; color decisions are
deferred until color output is implemented.

Example:

```text
tuck pkg use zsh   (context: home, dry-run)

plan:
  + mkdir  ~/.config/zsh
  + link   ~/.config/zsh/.zshrc -> ~/.dotfiles/zsh/.config/zsh/.zshrc

2 actions, 0 conflicts
re-run with --apply to execute
```

Error format:

```text
error: <message>
hint: <actionable suggestion>
```

### 9.2 JSON envelope

With `--json`, a command prints exactly one JSON document:

```json
{
  "schemaVersion": 1,
  "command": "package use",
  "context": "home",
  "kind": "plan",
  "data": {},
  "exitCode": 0
}
```

- `schemaVersion` -- integer; incremented on breaking changes.
- `command` -- canonical command name, including group when present.
- `context` -- `"home"` or `"root"` for domain commands; omitted or `"home"` for
  source/meta commands.
- `kind` -- one of `plan`, `sources`, `packages`, `tree`, `status`, `help`,
  `version`, `error`.
- `data` -- payload determined by `kind`.
- `exitCode` -- process exit code (`0` or `1`).

#### 9.2.1 `kind: "plan"`

Used by `adopt`, `eject`, `package use`, `package drop`, and
`package refresh`.

```json
{
  "schemaVersion": 1,
  "command": "package use",
  "context": "home",
  "kind": "plan",
  "data": {
    "dryRun": true,
    "applied": false,
    "packages": ["public:home:zsh"],
    "privilege": { "required": false },
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

Conflicts are included in the plan data and make `exitCode` `1`.

#### 9.2.2 `kind: "packages"`

Emitted by `package list`:

```json
{
  "schemaVersion": 1,
  "command": "package list",
  "context": "home",
  "kind": "packages",
  "data": {
    "source": "public",
    "packages": ["git", "zsh"]
  },
  "exitCode": 0
}
```

#### 9.2.3 `kind: "tree"`

Emitted by `package show`:

```json
{
  "schemaVersion": 1,
  "command": "package show",
  "context": "home",
  "kind": "tree",
  "data": {
    "package": {
      "identity": "public:home:zsh",
      "root": "/home/me/.dotfiles/zsh",
      "entries": [
        { "rel": ".config/zsh", "type": "dir" },
        { "rel": ".config/zsh/.zshrc", "type": "leaf" }
      ]
    }
  },
  "exitCode": 0
}
```

#### 9.2.4 `kind: "status"`

Emitted by `status` and `package status`:

```json
{
  "schemaVersion": 1,
  "command": "package status",
  "context": "home",
  "kind": "status",
  "data": {
    "entries": [
      {
        "targetPath": "/home/me/.config/zsh/.zshrc",
        "state": "deployed",
        "package": "public:home:zsh",
        "entry": "/home/me/.dotfiles/zsh/.config/zsh/.zshrc"
      }
    ]
  },
  "exitCode": 0
}
```

`state` is one of `deployed`, `absent`, `conflict`, `mismatch`,
`owned_by_other`, or `unmanaged`.

#### 9.2.5 `kind: "sources"`

Emitted by `source list`, `source add`, `source rm`, and `source default`:

```json
{
  "schemaVersion": 1,
  "command": "source list",
  "kind": "sources",
  "data": {
    "sources": [
      { "id": "public", "path": "/home/me/.dotfiles", "enabled": true, "default": true }
    ]
  },
  "exitCode": 0
}
```

#### 9.2.6 `kind: "help"` and `kind: "version"`

`tuck --help --json` emits command metadata. `tuck --version --json` emits
version metadata. There is no `help` command:

```json
{
  "schemaVersion": 1,
  "command": "tuck",
  "kind": "help",
  "data": { "...": "..." },
  "exitCode": 0
}
```

```json
{
  "schemaVersion": 1,
  "command": "version",
  "kind": "version",
  "data": { "version": "dev" },
  "exitCode": 0
}
```

#### 9.2.7 `kind: "error"`

```json
{
  "schemaVersion": 1,
  "command": "package use",
  "context": "home",
  "kind": "error",
  "data": {
    "error": {
      "code": "package_not_found",
      "message": "package \"ssh\" not found in source \"public\"",
      "hint": "pass --source private if it lives there, or check `tuck pkg list`",
      "details": { "ref": "ssh", "source": "public", "context": "home" }
    }
  },
  "exitCode": 1
}
```

---

## 10. Exit codes and error codes

Exit codes are binary:

| Code | Meaning |
| --- | --- |
| `0` | Success: command completed, dry-run printed, read completed, or help/version printed. |
| `1` | Failure: CLI parse/dispatch issue, config/state problem, resolution error, conflict, privilege failure, or runtime error. |

Detailed failure classification is carried by stderr and, with `--json`, by
`data.error.code`.

Stable error codes include:

- configuration/state: `manifest_missing`, `manifest_invalid`,
  `state_invalid`, `source_root_missing`, `no_source`, `unknown_source`;
- references/resolution: `invalid_ref`, `package_not_found`;
- target classification/conflicts: `real_file`, `real_directory`,
  `special_file`, `unmanaged_symlink`, `owned_by_other`, `path_mismatch`,
  `multiple_providers`, `outside_target_root`, `inside_source_repo`,
  `package_path_exists`, `not_a_managed_symlink`, `copy_drift`,
  `copy_source_modified`, `copy_target_modified`;
- state validation: `state_checksum_mismatch`;
- execution: `privilege_required`, `io_error`.

`package status` exits `0` when the query succeeds, even if it reports conflicts
inside the response body.

---

## 11. Help and usage text

`tuck --help` and per-command help are rendered by urfave/cli's default help
templates from command metadata. Help output includes the command name, usage
line, version where relevant, visible commands, and options. Exact section order,
capitalization, spacing, and flag rendering are framework-owned.

Acceptance tests assert help loosely with exit code and stable key substrings,
not byte-for-byte output. `--help --json` is the stable machine-readable help
surface.

---

## 12. Resolution algorithms

This section is normative. It specifies the internal resolution, ownership,
conflict, and operation algorithms that the command reference and plan model
build on.

### 12.1 Contexts, bases, and identities

| Context | `packageBase(source, context)` | `targetRoot(context)` |
| --- | --- | --- |
| `home` | `<source.path>` | `$HOME` |
| `root` | `<source.path>/.root` | `/` |

- A package base is the directory that holds packages for one source and
  context.
- A package root is one concrete package directory inside a base:
  `packageRoot(source, context, name) = join(packageBase(source, context), name)`.
- A package identity is `source-id + context + package-name`, displayed as
  `source:context:name`.
- A managed entry is a target symlink whose payload resolves inside the active
  source's package root and whose target location matches the package-relative
  path.

### 12.2 Path primitives

#### Expand input path

For any user-supplied path: expand a leading `~`; if relative, resolve against
the process current working directory; clean lexical components; do not follow
the final component when the command must inspect the symlink itself (`adopt`,
`eject`, `status`).

#### Canonicalize source roots

For each enabled source, expand `~`, make absolute, clean, and resolve symlinks
in the root itself. A source root that does not exist is a state error.

#### Check containment

`inside(child, root)` is true only when `child` equals `root` or is a descendant
after both are absolute and clean. The test is path-segment aware:
`/home/me/.dotfiles-private` is not inside `/home/me/.dotfiles`.

#### Convert package path to target path

```text
rel = relativePath(packageRoot, packageEntryPath)
reject if rel is "." or starts with ".."
targetPath = clean(join(targetRoot, rel))
reject if not inside(targetPath, targetRoot)
return targetPath
```

#### Convert target path to package path

```text
absTarget = expand input path
reject if not inside(absTarget, targetRoot)
rel = relativePath(targetRoot, absTarget)
reject if rel is "." or starts with ".."
packagePath = clean(join(packageRoot, rel))
reject if not inside(packagePath, packageRoot)
return packagePath
```

For `root`, `targetRoot` is `/`; root commands must still reject paths inside
any enabled source repository to avoid adopting a source into itself.

### 12.3 Source and package resolution

#### Select active source

```text
if --source given:
    look up enabled source by id            (else unknown_source)
else if a machine-local default source is set:
    use it
else if exactly one source is enabled:
    use it
else:
    error no_source
```

`eject` and `status` select the active source the same way and infer ownership
within it.

#### Parse package reference

Reject before resolution: any `:`, an empty name, a name starting with `.`, an
absolute name, a name with `..` segments, or a name containing a path separator.

#### Resolve existing package

For `package use`, `package drop`, `package refresh`, `package show`, and
`package status <ref>`:

```text
parse ref
source = select active source
packageRoot = packageRoot(source, context, name)
if packageRoot does not exist: error package_not_found
return source, context, name, packageRoot
```

#### Resolve package for adopt

`adopt` may create a new package path, but the source is fixed:

```text
parse ref
source = select active source
packageRoot = packageRoot(source, context, name)   # may or may not exist
return source, context, name, packageRoot
```

### 12.4 Package entry enumeration

```text
walk packageRoot depth-first
for each entry (skip packageRoot itself):
    rel = relativePath(packageRoot, entry)
    reject if rel escapes packageRoot
    if entry is a directory: record a directory entry
    else:                    record a leaf entry
```

Directory entries cause real directories to be created in the target tree and
are never represented as symlinks. Leaf entries are linked. If a package leaf is
itself a symlink, the target link points at the package symlink itself.

Package/file metadata may override a leaf's deployment strategy. The default
strategy is `symlink`; `copy` materializes the package file as a separate target
file and records state for future ownership/drift checks. Directories never use
`copy`.

### 12.5 Ownership resolution

#### Classify target path

Input: `targetPath`, optional selected package identity, context, and active
source.

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

Ownership inference scans only the active source.

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

Broken symlinks are still classifiable if their lexical target is inside a
package root. A managed symlink whose link path does not match its
package-relative path is reported as a mismatch and is never mutated
automatically.

#### Classify copied target

Copied-file ownership is state-backed, not inferred from filesystem structure:

```text
stat = lstat(targetPath)
if not exists: return Absent
if not regular file: return conflict

record = lookup copy state by active source, context, package, packageRel
if no record for targetPath: return UntrackedCopyTarget

sourceChecksum = checksum(packageEntryPath)
targetChecksum = checksum(targetPath)
sourceChanged = sourceChecksum != record.sourceChecksum
targetChanged = targetChecksum != record.targetChecksum

if sourceChanged and targetChanged: return CopyBothModified
if sourceChanged:                   return CopySourceModified
if targetChanged:                   return CopyTargetModified
return TrackedCopyUnchanged
```

Copy state is used only for entries configured with `deploy = "copy"`. Symlink
ownership remains payload-inferred.

### 12.6 Conflict rules

**Package use (leaf).** A leaf target is linkable when it is absent, or already a
symlink owned by the selected package pointing at the same entry. It conflicts
when it is a real file, real directory, special file, unmanaged symlink, managed
by another package, or managed by the selected package but mapping to a different
package-relative path.

**Package use (copy leaf).** A leaf configured with `deploy = "copy"` is
copyable when the target is absent, or when it is already tracked as a copied
entry and neither the source nor target has drifted from the recorded checksums.
It conflicts when the target exists but is untracked, when a tracked target has
been modified, when the package source has changed without an explicit refresh,
or when any non-regular-file condition would make copying unsafe. Applying the
copy updates the copied-entry state checksums.

**Directory.** A package directory entry's target is valid when absent or already
a real directory. It conflicts when the target is a file, symlink, or
non-directory special file.

**Adopt.** Requires: target exists, is a real file, is inside the selected target
root, is not inside any enabled source repository, and destination package path
does not already exist.

**Eject.** Requires: target is a symlink managed in the selected context, managed
package file exists and is not a directory, symlink path matches the
package-relative target path, and materializing the file does not overwrite
unrelated content. After moving the package file back to the target, eject
removes empty intermediate directories along the source/package path, stopping
before the package root.

### 12.7 Operation algorithms

Every mutating command first builds a complete plan. If any conflict is found,
it prints conflicts and mutates nothing. All are dry-run by default and mutate
only with `--apply`.

#### `package use`

```text
resolvedPackages = resolve existing package for each ref (or all packages)
plannedTargets = {}

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
        if entry.deploy == "copy":
            switch classifyCopiedTarget(targetPath, entry):
                Absent:                      plan copy entry -> targetPath
                TrackedCopyUnchanged:         no-op
                else:                        conflict
            continue
        switch classify(targetPath, selected package):
            Absent:                                plan symlink targetPath -> entry
            ManagedBySelectedPackage(same entry):  no-op
            else:                                  conflict
```

Symlink payloads are relative:
`payload = relativePath(dirname(targetPath), packageEntryPath)`.

#### `package drop`

```text
resolvedPackages = resolve existing package for each ref
for each package, for each leaf entry:
    targetPath = convert package path -> target path
    if entry.deploy == "copy":
        switch classifyCopiedTarget(targetPath, entry):
            Absent:                      no-op
            TrackedCopyUnchanged:         plan remove_copy targetPath
            else:                        conflict
        continue
    switch classify(targetPath, selected package):
        Absent:                                no-op
        ManagedBySelectedPackage(same entry):  plan remove_symlink targetPath
        else:                                  conflict
```

Directories are never pruned.

#### `package refresh`

```text
build drop plan for selected packages; if conflicts: stop
build use plan against the post-drop state; if conflicts: stop
apply all remove_symlink/remove_copy actions, then all symlink/copy/mkdir actions
```

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
plan symlink targetPath -> packagePath
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
for dir in parents(dirname(packagePath), stop before owner.packageRoot), deepest first:
    if dir will be empty after the move and earlier planned rmdir actions:
        plan rmdir dir
```

The package root itself is left in place. Removing a package is a separate
operation.

### 12.8 Listing algorithms

#### `package list`

```text
base = packageBase(activeSource, context)
list direct child directories of base as packages
    skip names starting with `.`
    skip non-directories
```

#### `package show`

Resolve the package in the active source and show that package root's tree.

#### `package status`

With a ref, enumerate package leaf entries and classify each target path. Without
a ref, summarize every package in the active source/context.

### 12.9 Execution planning

Mutations are explicit actions: `mkdir`, `symlink`, `remove_symlink`, and
`move`. Planning resolves the active source, packages, target paths, and
ownership before mutating; accumulates all conflicts; exits `1` without mutation
on any conflict; prints actions on a clean plan; and mutates only with `--apply`.
Root-context mutations make their privilege requirement visible in the plan and
never self-escalate.

---

## Appendix A -- Worked examples

### A.0 Bootstrap a new machine

```text
$ git clone git@github.com:me/dotfiles.git ~/.dotfiles
$ tuck source add ~/.dotfiles --default
added source "public" -> /home/me/.dotfiles (default)

$ tuck source list
* public   /home/me/.dotfiles          (default)

$ tuck pkg use zsh git --apply
2 actions, 0 conflicts -- applied
```

### A.1 Package use

```text
$ tuck pkg use zsh
tuck pkg use zsh   (context: home, dry-run)

plan:
  + mkdir  ~/.config/zsh
  + link   ~/.config/zsh/.zshrc -> ~/.dotfiles/zsh/.config/zsh/.zshrc

2 actions, 0 conflicts
re-run with --apply to execute
```

### A.2 Adopt a file

```text
$ tuck adopt ~/.gitconfig git
plan:
  ~ move   ~/.gitconfig -> ~/.dotfiles/git/.gitconfig
  + link   ~/.gitconfig -> ~/.dotfiles/git/.gitconfig

$ tuck adopt ~/.gitconfig git --apply
applied
```

### A.3 Eject a file

```text
$ tuck eject ~/.gitconfig --apply
plan:
  - unlink ~/.gitconfig
  ~ move   ~/.dotfiles/git/.gitconfig -> ~/.gitconfig
applied
```

### A.4 Status

```text
$ tuck status ~/.gitconfig
deployed by public:home:git

$ tuck pkg status git
git: 1 deployed, 0 absent, 0 conflicts
```

### A.5 Root context

```text
$ tuck pkg use sshd --root
# dry-run by default

$ tuck pkg use sshd --root --apply
error: root-context write requires elevated privileges
hint: re-run with sudo
# exit 1; with --json, error.code is privilege_required
```

### A.6 Copy deployment for symlink-hostile files

Some applications reject symlinked config files. Configure those package leaves
in repo `tuck.toml`:

```toml
[package.zsh]

[[package.zsh.file]]
path = ".config/symlink-hostile-app/config"
deploy = "copy"
mode = "0600"
```

Then `package use` plans a copy rather than a symlink:

```text
$ tuck pkg use zsh
plan:
  + copy   ~/.dotfiles/zsh/.config/symlink-hostile-app/config -> ~/.config/symlink-hostile-app/config (mode 0600)

1 action, 0 conflicts
```

Copied targets are tracked in machine-local state. If the target is edited after
deployment, later `package use`, `package drop`, or `package refresh` reports
copy drift instead of overwriting or removing it silently.

---

## Appendix B -- Relationship to the previous CLI surface

This spec redesigns the CLI surface from first principles while keeping the
domain model fixed.

Deliberate changes:

1. **Command depth mirrors operation frequency.** File operations are top-level,
   package operations are grouped under `package`/`pkg`, and rare source
   operations are grouped under `source`.
2. **Package verbs changed.** `deploy`/`undeploy`/`redeploy` became
   `package use`/`package drop`/`package refresh`. A package is a collection of
   files; it does not itself get "linked".
3. **File operations are file-first.** `adopt <pkg> <file>` became
   `adopt <file> <pkg>`. The file is the subject of the command.
4. **Status split by level.** File status is `tuck status <file>`. Package status
   is `tuck package status [pkg]`.
5. **Source commands aligned with package commands.** `source enable` became
   `source add`; `source rm` and `source default` are part of the designed
   surface.
6. **No false globals.** `--source`, `--root`, and `--apply` are scoped to the
   commands where they matter. `--json` and `--no-color` are universal.
7. **Exit codes are binary.** Detailed classification moved from process exit
   codes into stderr and the JSON `error.code`.
8. **Machine-readable meta output.** `--help --json` and `--version --json` are
   supported for tooling; `help` is not a command.

The internal algorithms for path primitives, package enumeration, ownership
inference, conflict rules, and execution planning are preserved, translated into
the current vocabulary.

### Implementation framework

The CLI is built on urfave/cli v3. The framework is authoritative for surface
mechanics: command/flag parsing, exact flag placement behavior, help/version
rendering, and usage-error formatting. The spec aims to be close to, not
byte-identical with, framework defaults. Where they differ, prefer idiomatic
urfave/cli behavior when that is the intended product decision, then update this
spec and acceptance tests to match.
