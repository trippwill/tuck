# M21-M22 green: status

Status: historical milestone note. It describes the red/green state at the time
M21-M22 were implemented and may mention APIs or failing behavior that have since
been refactored.

## Problem and approach

Implement MVP Vertical C from `docs/backlog.md`: active-source ownership
inference and read-only status commands. The slice should extract reusable
package/target classification logic from the `package use` implementation, then
wire:

- `tuck status <file>` for one target path;
- `tuck package status <package-ref>` for one package's leaf entries;
- `tuck package status` for the spec's all-package status summary.

This is a read-only slice. It should not mutate the target tree, package source,
or machine state.

## Normative references

Match `docs/cli-spec.md` unless this spec explicitly narrows scope:

- section 7.3 `status`;
- section 7.9 `package status`;
- section 9.2.4 `kind: "status"`;
- section 10 exit and error codes;
- section 12.1-12.6 context, package identity, path, ownership, and conflict
  algorithms.

## Scope

In scope:

- `home` context status.
- Active-source-only symlink ownership inference.
- Symlink deployment status: deployed, absent, conflict, mismatch,
  owned_by_other, unmanaged.
- `package status` with zero or one package ref.
- Human and JSON status output.
- Reuse by `package use` of extracted classification logic, preserving Vertical B
  behavior.

Out of scope:

- `root` context behavior beyond an explicit not-implemented failure until M26.
- Copy deployment status and drift detection.
- `package list`, `package show`, `package drop`, `adopt`, and `eject`, except
  for shared primitives that they will later reuse.
- Color behavior beyond respecting existing `--no-color`/`--json` conventions.

No new exported variable sentinels should be added. Use generated app-error
helpers or unexported typed constants where errors need stable identity.

## Current implementation state

- `status` and `package status` command metadata exists in `internal/app`, but
  both actions are stubs.
- `internal/plan/use.go` already has private package resolution, entry
  enumeration, target classification, and symlink owner inference for `package
  use`.
- `internal/pkgref` and `internal/pathutil` are available and should be reused.
- JSON envelope infrastructure exists in `internal/app/output.go`.

## Engine design

Introduce or extract focused read-only engine packages:

- `internal/packages`
  - package identity: `source:context:name`;
  - direct package discovery for one source/context;
  - existing package resolution for one/many/all refs;
  - deterministic entry enumeration with directory/leaf distinction.
- `internal/target`
  - active-source-only symlink owner inference;
  - target classification for absent, real file, real directory, special file,
    unmanaged symlink, managed symlink, selected package, other package, and
    mismatch.
- `internal/status`
  - file status query;
  - package status query;
  - all-package status summary;
  - JSON/human-facing status data structs.

Avoid making read-only status depend on mutation planning. `internal/plan` should
use the shared package/target APIs instead.

## Status mapping

Use these states in JSON and human output:

| State | Meaning |
| --- | --- |
| `deployed` | Target is a managed symlink in the active source and maps to the expected package-relative path. |
| `absent` | Target path does not exist. |
| `conflict` | Target is a real file, real directory, special file, or otherwise blocks the selected package entry. |
| `mismatch` | Symlink points inside the active source, but the link path does not match the package-relative target. |
| `owned_by_other` | Package status is checking package B, but the target is managed by package A in the active source. |
| `unmanaged` | Target is a symlink that does not point inside the active source package base. |

For `status <file>`, a managed active-source symlink is `deployed` because there
is no selected package. `owned_by_other` is primarily a package-status state.

## JSON output

Emit one envelope:

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

The CLI-spec fields above are required. Optional fields such as `code`,
`message`, `owner`, or `expectedTarget` may be added when they clarify
non-deployed states, but keep names stable once introduced.

`package status` exits `0` when the query succeeds even if entries contain
conflict-like states. Resolution and usage errors still exit `1`.

## Human output

Keep human output concise and deterministic:

- header with command, context, and source;
- one line per status entry: state, target path, package identity when known, and
  code/message for non-deployed states;
- final summary count.

Primary status output goes to stdout. Error diagnostics and hints go to stderr.
With `--json`, stdout must contain exactly one envelope and stderr must be empty
for handled errors.

## Acceptance coverage

Add package-status scripts under `acceptance/testdata/script/package/` and
top-level status scripts under `acceptance/testdata/script/target/`.

Suggested scripts:

1. `file-managed.txtar` - apply a package with `package use`, then classify the
   target with `status`; expect `deployed`.
2. `file-absent-unmanaged-mismatch.txtar` - classify absent path, unmanaged
   symlink, and active-source mismatch.
3. `file-json.txtar` - assert one `kind:"status"` envelope for `status <file>`.
4. `package-status-mixed.txtar` - package entries report deployed, absent, and
   conflict states in one successful command.
5. `package-status-owned-by-other.txtar` - two packages share a target; querying
   the non-owner reports `owned_by_other`.
6. `package-status-summary.txtar` - `package status` with no ref summarizes all
   packages in deterministic order.
7. `package-status-json.txtar` - assert package status JSON envelope and entries.
8. `status-errors.txtar` - invalid ref, missing package, no active source, and
   root-not-implemented cases.
9. `usage.txtar` - top-level `status` requires exactly one path; package status
   accepts zero or one ref.

## Unit coverage

- Ownership inference:
  - relative and absolute payloads;
  - broken symlink whose lexical target is inside the source;
  - symlink into another source is unmanaged;
  - path mismatch reports expected target.
- Target classification:
  - absent, real file, real directory, unmanaged symlink, managed symlink,
    managed by selected package, managed by other package, mismatch.
- Package extraction:
  - discovery order;
  - skip `.root`;
  - nested leaf enumeration;
  - package symlink entries are leaves.
- Status builders:
  - file status has one entry;
  - package status reports every leaf;
  - no-ref summary aggregates all packages deterministically;
  - conflict-like body states do not make the command fail.
- Rendering:
  - human output has stable substrings;
  - JSON envelope matches `kind:"status"` contract.

## Milestones

### M21 - Infer active-source symlink ownership

- Extract shared package identity and entry enumeration.
- Extract target classification and owner inference.
- Convert `package use` to the shared classification API.
- Keep existing package-use tests green.

### M22 - Implement status commands

- Add file/package/all-package status builders.
- Add human and JSON status renderers.
- Wire `status` and `package status` CLI actions and usage validation.
- Add acceptance coverage.
- Mark M21-M22 complete in `docs/backlog.md` only after the full gate is green.

## Verification

Focused checks:

```sh
go test ./internal/... ./cmd/...
go test -tags tuck_testhooks -run 'TestSuites/(package|target)' ./acceptance/...
```

Final gate:

```sh
mise run check
git diff --check
```
