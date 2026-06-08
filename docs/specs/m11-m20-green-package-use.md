# M11-M20 green: package use for home

## Problem and approach

Implement MVP Vertical B from `docs/backlog.md` M11-M20: make `tuck package use`
work end-to-end for the default `home` context. The slice should parse package
refs, resolve packages in the active source, enumerate package entries, classify
targets, build a complete dry-run plan with accumulated conflicts, render human
and JSON plan output, apply conflict-free symlink/mkdir plans only with
`--apply`, and wire the CLI command.

The source of truth is `docs/cli-spec.md`, especially:

- §3.1/§3.2 context and source selection;
- §4 flag interactions (`package use` requires refs or `--all`, but not both);
- §6 package references;
- §7.4 `package use`;
- §8 and §12.7 planning/action algorithms;
- §9 plan output envelopes and stream rules;
- §10 error codes.

Scope this slice deliberately:

- Implement `home` context package use only. The root context is M26. If
  `--root` reaches the new action before M26, fail explicitly rather than
  silently treating it as home.
- Implement symlink deployment only. Manifest package metadata and `deploy =
  "copy"` are First Release work.
- Do not implement `package drop`, `refresh`, `list`, `show`, `status`, `adopt`,
  or `eject` beyond shared primitives needed by package use.
- Keep package refs as plain package names: no source/context/path encoding.
- Preserve the output contract: primary results and JSON on stdout; diagnostics
  and hints on stderr; with `--json`, exactly one stdout envelope and empty
  stderr.

## Current state

- CLI skeleton exists in `internal/app` with `package use` currently
  `notImplemented`.
- Source registry and active source resolution are implemented:
  `state.Load()` and `resolve.ActiveSource(registry, explicitID)`.
- Generated `apperr` helpers are available. New engine packages should define
  string sentinel types with `//go:generate go run ../../cmd/errgen -types ...`.
- `internal/pathutil` is still a scaffold and is available for path-segment-aware
  primitives.
- Acceptance harness has `wanthome` and `readlink`; no package-use suite exists.
- `mise run check` now runs generation, command tests, unit tests, acceptance,
  build, vet, and fmt.

## Proposed package layout

Use focused internal packages so algorithmic logic stays out of `internal/app`.
Names can shift during implementation if one package becomes too broad, but keep
the boundaries:

- `internal/pkgref`
  - parse/validate package references;
  - errors: `ErrInvalidRef`.
- `internal/pathutil`
  - clean/absolute containment checks;
  - path-segment-aware `Inside`;
  - relative path and relative symlink payload helpers;
  - target/package path conversion helpers that accept explicit roots.
- `internal/pkgsrc` or `internal/packages`
  - target context representation (`home` now; root later);
  - package identity (`source:context:name`);
  - package base/root resolution;
  - existing-package lookup and `--all` package discovery;
  - package entry enumeration (`directory` vs `leaf`) with deterministic order.
- `internal/target`
  - classify target paths with `lstat`;
  - infer symlink ownership within the active source only;
  - detect managed-by-selected, managed-by-other, mismatch, unmanaged symlink,
    real file/dir, special file, absent.
- `internal/plan`
  - action and conflict model;
  - build package-use plans;
  - apply conflict-free mkdir/symlink actions.

If splitting that finely creates churn, combine `pkgsrc`, `target`, and `plan`
behind clear files in a single `internal/plan` package, but keep ref/path helpers
separate because they will be reused by later verticals.

## Red/green test strategy

Follow the repo rule: red tests must compile and fail for behavior, not missing
symbols. Add minimal production compile seams before tests that need new APIs.

Acceptance suite first:

- Add `acceptance/package_use_test.go` with `TestPackageUse`.
- Scripts under `acceptance/testdata/script/package_use/`.
- Use `wanthome` to create `$HOME`, state, and source manifest.
- Use inline txtar files under `src/<package>/...` for package contents.
- Assert dry-run mutates nothing, `--apply` mutates exactly as planned, symlink
  payloads are relative using `readlink`, stderr is empty on success, and failures
  classify through stderr/JSON.

Suggested red scripts:

1. `dry-run.txtar`
   - `wanthome`;
   - source package `src/zsh/.config/zsh/.zshrc`;
   - `exec tuck pkg use zsh --no-color`;
   - assert stdout includes a plan and "re-run with --apply";
   - assert `$HOME/.config/zsh/.zshrc` does not exist.
2. `apply.txtar`
   - same package;
   - `exec tuck pkg use zsh --apply --no-color`;
   - assert directories and symlink exist;
   - assert payload is relative from target dir to source entry.
3. `json.txtar`
   - dry-run with `--json`;
   - assert one plan envelope with `kind:"plan"`, `context:"home"`,
     `dryRun:true`, `applied:false`, package identity, mkdir/symlink actions,
     and empty conflicts.
4. `missing-package.txtar`
   - `! exec tuck pkg use missing --no-color`;
   - assert `package_not_found`-style diagnostic/hint.
5. `invalid-ref.txtar`
   - refs with `/`, `..`, `:`, or absolute paths fail before package resolution
     with `invalid_ref`.
6. `conflicts.txtar`
   - existing real file and existing real directory where leaves/dirs need
     different types;
   - assert all conflicts are reported and no mutation occurs.
7. `multiple-providers.txtar`
   - two packages in one invocation map to the same target leaf;
   - assert `multiple_providers` conflict and no mutation.
8. `idempotent.txtar`
   - target already has the correct managed symlink;
   - assert no symlink action is planned and apply is a no-op success.
9. `all.txtar`
   - `tuck pkg use --all` discovers package dirs in deterministic order, skipping
     `.root` and non-package entries.
10. `usage.txtar`
    - no refs without `--all` fails;
    - refs plus `--all` fails.

Unit tests by milestone:

- M11: package ref parser table tests.
- M12: path containment, package-to-target conversion, target-to-package
  conversion, relative symlink payloads, path-segment edge cases.
- M13: package discovery/enumeration with deterministic ordering, nested dirs,
  leaf symlinks treated as leaves, skip `.root`/`tuck.toml` for package discovery.
- M14: active-source + package resolution, `package_not_found`, context base.
- M15: target classification and symlink ownership inference, including broken
  managed symlinks and path mismatch.
- M16/M17: complete plan building with conflict accumulation, deduped duplicate
  refs, multiple providers, directory/leaf conflicts, idempotent managed links.
- M18: human and JSON rendering tests around the stable output shape.
- M19: apply tests for mkdir/symlink ordering, dry-run no mutation, conflict no
  mutation.

## Milestone implementation plan

### M11 Parse and validate package refs

- Add `internal/pkgref`.
- Define generated error helpers with `ErrRef` or similar.
- `Parse(ref string) (Ref, error)` rejects:
  - empty/whitespace-only;
  - `:`;
  - absolute paths;
  - any path separator;
  - `..` path segments.
- Preserve the original package name string for display and path construction.
- Add `invalid_ref` classification in `internal/app/output.go`.

### M12 Path primitives

- Flesh out `internal/pathutil`.
- Implement path-segment-aware `Inside(child, root)`.
- Implement relative path helpers that reject escapes.
- Implement package-entry-to-target conversion:
  - `rel = relative(packageRoot, packageEntryPath)`;
  - reject `"."` and escapes;
  - `targetPath = clean(join(targetRoot, rel))`;
  - reject target escaping target root.
- Implement target-to-package conversion for future adopt/eject reuse.
- Implement relative symlink payload:
  `relativePath(dirname(targetPath), packageEntryPath)`.
- Keep functions parameterized by roots so tests do not touch real `$HOME`.

### M13 Enumerate packages and entries

- Add package base and entry enumeration:
  - package base for home is `source.Path`;
  - package root is `join(base, ref.Name)`.
- Discover `--all` packages by direct child directories of the package base:
  - skip `.root`;
  - skip `tuck.toml`;
  - skip non-directories;
  - sort names lexically for deterministic plans.
- Enumerate a selected package root depth-first:
  - skip the package root itself;
  - record directories as directory entries;
  - record everything else, including symlinks, as leaf entries;
  - sort deterministically.

### M14 Resolve existing packages

- Compose `state.Load()` and `resolve.ActiveSource()` with package ref parsing.
- Resolve one or more refs to package identities:
  `sourceID:home:packageName`.
- Missing package root returns package resolution error `package_not_found`.
- Duplicate refs in one invocation are deduped while preserving first-seen order.
- `--all` resolves all discovered packages.

### M15 Classify target paths and infer ownership

- Add `internal/target` or equivalent.
- Classify with `os.Lstat`:
  - absent;
  - real directory;
  - real file;
  - special file;
  - symlink.
- For symlinks:
  - read payload;
  - resolve relative payload against link parent lexically;
  - infer ownership only if resolved target is inside active source package base;
  - derive package name and package-relative path;
  - compute expected target path from package-relative path;
  - return mismatch if symlink path does not equal expected target path;
  - compare selected package identity when provided.
- Broken managed symlinks should still be classifiable by lexical payload.

### M16 Detect package-use and directory conflicts

- Implement conflict codes needed by package use:
  - `real_file`;
  - `real_directory`;
  - `special_file`;
  - `unmanaged_symlink`;
  - `owned_by_other`;
  - `path_mismatch`;
  - `multiple_providers`.
- Directory entries:
  - absent -> mkdir action;
  - real directory -> no-op;
  - anything else -> conflict.
- Leaf entries:
  - absent -> symlink action;
  - managed by selected package for same entry -> no-op;
  - anything else -> conflict.
- Accumulate every conflict before returning.

### M17 Build complete action plan

- Define `Plan`, `Action`, `Conflict`, and `PackageIdentity`.
- Plan includes:
  - command (`package use`);
  - context (`home`);
  - dry-run/applied flags;
  - selected package identities;
  - ordered actions;
  - conflicts;
  - privilege marker (`required:false` for home).
- Build package-use plan from resolved packages without mutating.
- Ensure deterministic order:
  - package order from refs or sorted `--all`;
  - directories before leaves per package, or stable walk order if rendering is
    clearer;
  - parent directories before child entries.

### M18 Render plan output

- Extend `internal/app/output.go` with plan rendering.
- Human output:
  - header: `tuck package use ... (context: home, dry-run|apply)`;
  - `plan:` block with mkdir/link actions;
  - optional `conflicts:` block;
  - summary `<n> actions, <m> conflicts`;
  - dry-run hint when conflict-free and not applied.
- JSON output:
  - one envelope with `kind:"plan"`;
  - include `context:"home"`;
  - include `dryRun`, `applied`, `packages`, `privilege`, `actions`,
    `conflicts`;
  - conflicts use exitCode `1`, stderr empty.
- Decide and document any small output-shape choices not already pinned by
  `docs/cli-spec.md`.

### M19 Apply conflict-free plans only with `--apply`

- If plan has conflicts: render plan/conflicts, exit `1`, mutate nothing.
- If no conflicts and no `--apply`: render dry-run plan, mutate nothing, exit `0`.
- If no conflicts and `--apply`:
  - create directories with `0755` subject to umask;
  - create symlinks with relative payloads;
  - skip no-op entries;
  - fail as `io_error` on filesystem errors.
- Apply in listed order.
- Do not overwrite existing files/dirs/symlinks outside the explicit conflict-free
  cases.

### M20 Wire `package use`

- Replace `notImplemented("package use")` with real action.
- Keep `package` alias `pkg` working.
- Validate `--all`/refs interaction in action:
  - no refs and no `--all` -> usage/domain error;
  - refs plus `--all` -> usage/domain error.
- Use urfave argument support where it helps, but `--all` plus variadic refs
  likely needs action-level validation.
- Use existing `--source`, `--root`, `--apply`, `--json`, `--no-color` flags.
- For Vertical B, fail explicitly on `--root` rather than silently applying home
  behavior.

## Error classification additions

Add stable classification for package-use errors:

- `invalid_ref`;
- `package_not_found`;
- conflict codes listed in M16;
- filesystem apply failures map to `io_error`.

Plan conflicts are not necessarily rendered through `renderError`; they belong
inside the plan/conflicts output and JSON plan data. Resolution errors before a
plan exists should use the error envelope/stderr diagnostic path.

## Backlog/docs updates

At completion:

- mark M11-M20 complete in `docs/backlog.md`;
- if output details are clarified beyond `docs/cli-spec.md`, update the spec in
  the same change.

## Verification

Red checkpoints:

```sh
go test ./internal/pkgref ./internal/pathutil
go test ./internal/... ./cmd/...
go test -tags tuck_testhooks -run TestPackageUse ./acceptance/...
```

Green/final:

```sh
mise run generate
mise run check
git diff --check
```

When validating behavior, always include:

- dry-run no mutation;
- conflict no mutation;
- apply creates relative symlinks;
- `--json` stdout-only envelope;
- `pkg` alias behaves like `package`.
