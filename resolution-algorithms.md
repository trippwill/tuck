# Dotfiles resolution algorithms

Final command name: `tuck`.

This document specifies resolution behavior for a dedicated dotfiles CLI that replaces the current `d.*` zsh aliases/functions without cloning all of GNU Stow. The CLI intentionally implements a narrower model:

- Package directories map onto real target directories.
- Only leaf entries are linked.
- Directory folding is never performed.
- The caller's current working directory must never affect Stow/link correctness, except when resolving an explicitly relative input path.

## CLI surface

The public CLI has two target contexts:

```text
tuck <command> ...        # user context; target root is $HOME
tuck root <command> ...   # root context; target root is /
```

There is no explicit `tuck user` command in the initial API. `user` is the internal name for the default target context.

Commands:

```text
link <package-ref>...
unlink <package-ref>...
relink <package-ref>...
capture <package-ref> <target-file>
release <target-link>
packages
tree [package-ref]
```

Command semantics:

- `link` creates managed target symlinks for package leaf entries.
- `unlink` removes managed target symlinks for selected packages. It does not move package files back into the target tree.
- `relink` refreshes selected package links by unlinking and linking them again.
- `capture` moves one existing real target file into a package, then creates a managed target symlink pointing back to it.
- `release` removes one managed target symlink and moves the package file back into the target tree.
- `packages` lists package directories.
- `tree` displays package contents.

The legacy `d.unlink` materialization workflow maps to `tuck release`, not `tuck unlink`.

Execution mode options for mutating commands:

- `--dry-run` prints the plan and performs no mutations.
- `--yes` allows `capture` and `release` to mutate after printing a conflict-free plan.

## Current behavior to preserve

Current user package context:

- Package source root: `~/.dotfiles`
- Target root: `$HOME`
- Link/restow/unstow aliases call GNU Stow with `--no-folding`

Current root package context:

- Package source root: `~/.dotfiles/.root`
- Target root: `/`
- Mutating operations use privileged execution
- Link/restow/unstow aliases call GNU Stow with `--no-folding`

Current special workflows:

- `d.adopt` works only on individual real files, not directories or symlinks.
- `d.adopt` creates the package path needed for Stow adoption, dry-runs by default, and only mutates with confirmation.
- `d.unlink` replaces a managed target symlink with the real file from the package. In this CLI, that behavior is named `release`.

## Core terms

### Source

A source is a configured dotfiles repository.

Example:

```toml
[sources.public]
path = "~/.dotfiles"
enabled = true

[sources.private]
path = "~/.dotfiles-private"
enabled = true
```

Each source has:

- `id`: stable name such as `public` or `private`
- `path`: repository path
- `enabled`: whether it participates in resolution

### Target context

A target context defines where package entries appear.

| Context | Package base | Target root |
| --- | --- | --- |
| user | `<source.path>` | `$HOME` |
| root | `<source.path>/.root` | `/` |

A package base is the directory that contains packages for one source and target context. A package root is one concrete package directory inside a package base. A target root is the filesystem root under which package entries appear.

The public CLI selects the root context with the `root` command prefix, for example:

```sh
tuck root link public:sshd
tuck root link sshd
tuck root capture private:ssh /etc/ssh/sshd_config
```

### Package identity

A package identity is:

```text
source id + target context + package name
```

Package identities are internal and display forms. CLI package refs do not include target context; the context comes from the command form (`tuck ...` or `tuck root ...`).

Examples:

```text
public:user:zsh
private:user:ssh
public:root:sshd
```

Package root:

```text
user package: <source.path>/<package>
root package: <source.path>/.root/<package>
```

### Managed entry

A managed entry is a target symlink whose resolved destination is inside a configured package root and whose target location matches the package-relative path.

Example:

```text
package file: ~/.dotfiles/zsh/.config/zsh/.zshrc
target link:  ~/.config/zsh/.zshrc
```

The target link payload may be relative or absolute. Ownership is inferred after resolving that payload to an absolute path.

## Path primitives

### Expand input path

For any user-supplied path:

1. Expand leading `~`.
2. If the path is relative, resolve it against the process current working directory.
3. Clean lexical components such as `.`.
4. Do not follow the final path component when the command needs to inspect a symlink itself, such as `release`.

### Canonicalize configured roots

For each enabled source and each target root:

1. Expand `~`.
2. Convert to absolute path.
3. Clean the path.
4. Resolve existing symlinks in the root itself.
5. Fail if an enabled source root does not exist, except where an explicit future config option allows creating it.

### Check containment

`inside(child, root)` returns true only when `child` is equal to `root` or is a descendant of `root` after both paths are absolute and clean.

This check must be path-segment aware:

```text
/home/me/.dotfiles-private is not inside /home/me/.dotfiles
```

### Convert package path to target path

Input:

```text
packageRoot
targetRoot
packageEntryPath
```

Algorithm:

```text
rel = relativePath(packageRoot, packageEntryPath)
reject if rel is "." or starts with ".."
targetPath = clean(join(targetRoot, rel))
reject if targetPath is not inside targetRoot
return targetPath
```

For `root`, `targetRoot` is `/`, so every absolute target path is technically inside the target root. Root commands must still reject paths inside configured source repositories to avoid adopting the dotfiles repo into itself.

### Convert target path to package path

Input:

```text
targetRoot
packageRoot
targetPath
```

Algorithm:

```text
absTarget = expand input path
reject if absTarget is not inside targetRoot
rel = relativePath(targetRoot, absTarget)
reject if rel is "." or starts with ".."
packagePath = clean(join(packageRoot, rel))
reject if packagePath is not inside packageRoot
return packagePath
```

## Source and package resolution

### Parse package reference

Package references are CLI input with this shape:

```text
[<source-id>:]<package-name>
```

They may be unqualified:

```text
zsh
ssh
```

or source-qualified:

```text
public:zsh
private:ssh
```

Package references never include target context. For example, `public:root:sshd` is an internal/display package identity, not valid CLI input.

Algorithm:

```text
if ref contains ":":
    reject if ref contains more than one ":"
    reject if ":" appears after any path separator
    sourceID = part before ":"
    packageName = part after ":"
else:
    sourceID = none
    packageName = ref

reject empty sourceID
reject empty packageName
reject sourceID containing a path separator
reject absolute packageName
reject packageName containing ".." path segments
reject packageName containing ":"
```

Package names should normally be direct children of the package base. If nested package names are allowed later, they must still be safe relative paths under the package base.

### Resolve package for existing-package operations

Existing-package operations:

```text
packages/tree for a package
link
unlink
relink
```

Input:

```text
context
packageRef
enabled sources
```

Algorithm:

```text
parse packageRef

if sourceID is explicit:
    source = enabled source with that id, else error
    packageRoot = packageRoot(source, context, packageName)
    if packageRoot does not exist: error "package not found"
    return source, context, packageName, packageRoot

candidates = []
for each enabled source:
    root = packageRoot(source, context, packageName)
    if root exists:
        add candidate

if candidates is empty:
    error "package not found"

if candidates has more than one item:
    error "ambiguous package"; show source-qualified candidates

return the only candidate
```

Default source ordering is useful for listing and display, but it must not silently break ties between public/private packages.

### Resolve package for capture

`capture` may create a new package path. Therefore an unqualified package that does not exist needs stricter handling.

Algorithm:

```text
parse packageRef

if sourceID is explicit:
    source = enabled source with that id, else error
    packageRoot = packageRoot(source, context, packageName)
    return source, context, packageName, packageRoot

candidates = enabled sources where packageRoot(source, context, packageName) exists

if candidates has one item:
    return it

if candidates has more than one item:
    error "ambiguous package"; require source-qualified package

if candidates is empty and exactly one source is enabled:
    return that source with a new packageRoot

if candidates is empty and multiple sources are enabled:
    error "new package requires source qualification"
```

## Package entry enumeration

Input:

```text
packageRoot
```

Algorithm:

```text
walk packageRoot depth-first

for each entry:
    if entry is packageRoot:
        continue

    rel = relativePath(packageRoot, entry)
    reject if rel escapes packageRoot

    if entry is a directory:
        record directory entry
        continue

    record leaf entry
```

Rules:

- Directory entries cause real directories to be created in the target tree.
- Directory entries are never represented as target symlinks.
- Leaf entries are linked into the target tree.
- Capture accepts only a single existing real target file.
- If package symlinks are supported as leaf entries, the target link should point to the package symlink itself, not to the package symlink's resolved destination. This keeps package ownership local to the package tree.

## Ownership resolution

### Classify target path

Input:

```text
targetPath
selected package identity, optional
context
enabled sources
```

Algorithm:

```text
stat = lstat(targetPath)

if targetPath does not exist:
    return Absent

if stat is directory and not symlink:
    return RealDirectory

if stat is regular file and not symlink:
    return RealFile

if stat is other non-symlink type:
    return SpecialFile

if stat is symlink:
    owner = inferSymlinkOwner(targetPath, context, enabled sources)
    if owner is ManagedPathMismatch:
        return ManagedPathMismatch(owner)
    if owner is none:
        return UnmanagedSymlink
    if selected package is not set:
        return ManagedSymlink(owner)
    if selected package is set and owner == selected package:
        return ManagedBySelectedPackage(owner)
    return ManagedByOtherPackage(owner)
```

### Infer symlink owner

Input:

```text
targetLinkPath
context
enabled sources
```

Algorithm:

```text
payload = readlink(targetLinkPath)

if payload is relative:
    targetAbs = clean(join(dirname(targetLinkPath), payload))
else:
    targetAbs = clean(payload)

for each enabled source:
    base = packageBase(source, context)
    if targetAbs is not inside base:
        continue

    relToBase = relativePath(base, targetAbs)
    packageName = first path segment of relToBase
    packageRoot = join(base, packageName)
    packageRel = relativePath(packageRoot, targetAbs)

    expectedTarget = clean(join(targetRoot(context), packageRel))

    if clean(targetLinkPath) != expectedTarget:
        return ManagedPathMismatch(source, context, packageName, packageRel, expectedTarget)

    return ManagedOwner(source, context, packageName, packageRoot, packageRel)

return none
```

Notes:

- Ownership is inferred from the symlink target path. No manifest is required.
- Broken symlinks can still be classified if their lexical target is inside a configured package root.
- A managed symlink whose link path does not match the package-relative path is not safe to mutate automatically. Report it as a managed path mismatch.

## Conflict rules

### Link conflicts

For a leaf package entry, the target path is linkable when:

- The target path is absent.
- The target path is already a symlink owned by the selected package and points to the same package entry.

The target path conflicts when:

- It is a real file.
- It is a real directory where a leaf link should be created.
- It is a special file.
- It is an unmanaged symlink.
- It is managed by another package.
- It is managed by the selected package but maps to a different package-relative path.

### Directory conflicts

For a package directory entry, the target directory is valid when:

- The target path is absent and can be created as a real directory.
- The target path already exists as a real directory.

The target directory conflicts when:

- The target path is a file.
- The target path is a symlink.
- The target path is any non-directory special file.

### Capture conflicts

Capture requires:

- The target path exists.
- The target path is a real file.
- The target path is not a symlink.
- The target path is not a directory.
- The target path is inside the selected target root.
- The target path is not inside any configured source repository.
- The destination package path does not already exist.

### Release conflicts

Release requires:

- The target path exists as a symlink.
- The symlink is managed in the selected context.
- The managed package file exists.
- The symlink path matches the package-relative target path.
- Replacing the symlink with the real file does not overwrite unrelated content.

## Operation algorithms

Every mutating command should first build a complete plan. If any conflict is found, the command should print the conflicts and perform no mutations.

### `link <package...>`

Input:

```text
context
package refs
```

Algorithm:

```text
resolvedPackages = resolve each package ref for existing-package operation
plannedTargets = empty map from target path to package entry

for each package:
    entries = enumerate package entries

    for each directory entry:
        targetDir = package path to target path
        classification = classify targetDir

        if classification is Absent:
            plan mkdir targetDir
        else if classification is RealDirectory:
            no-op
        else:
            conflict

    for each leaf entry:
        targetPath = package path to target path

        if targetPath already appears in plannedTargets with a different package entry:
            conflict "multiple packages provide same target"
            continue

        classification = classify targetPath with selected package

        if classification is Absent:
            plan symlink targetPath -> packageEntryPath
        else if classification is ManagedBySelectedPackage for the same package entry:
            no-op
        else:
            conflict
```

Symlink creation:

```text
payload = relativePath(dirname(targetPath), packageEntryPath)
create symlink with payload at targetPath
```

Relative symlink payloads preserve the GNU Stow-like behavior while ownership checks still resolve them to absolute paths.

### `unlink <package...>`

Input:

```text
context
package refs
```

Algorithm:

```text
resolvedPackages = resolve each package ref for existing-package operation

for each package:
    entries = enumerate package entries

    for each leaf entry:
        targetPath = package path to target path
        classification = classify targetPath with selected package

        if classification is Absent:
            no-op
        else if classification is ManagedBySelectedPackage for the same package entry:
            plan remove symlink targetPath
        else:
            conflict or report "not owned by selected package"
```

Directory pruning:

- Default behavior should not prune directories, because the tool does not track which real directories it created.
- A future `prune` or `doctor` command can safely remove empty directories only with explicit user intent.

### `relink <package...>`

`relink` refreshes selected package links using the same ownership rules.

Algorithm:

```text
build unlink plan for selected packages
if conflicts: stop

build link plan for selected packages against the post-unlink filesystem state
if conflicts: stop

apply unlink actions, then link actions
```

For an already-owned target symlink, `relink` may remove and recreate it so the symlink payload is normalized to the CLI's preferred relative form.

### `capture <package> <target-file>`

`capture` starts managing an existing real file.

Input:

```text
context
package ref
target-file
```

Algorithm:

```text
targetPath = expand target-file without following final symlink
reject if targetPath is outside targetRoot(context)
reject if targetPath is inside any configured source repository

classification = classify targetPath
reject unless classification is RealFile

package = resolve package ref for capture
packagePath = target path to package path

reject if packagePath already exists
reject if packagePath is outside package.packageRoot

plan mkdir dirname(packagePath)
plan move targetPath -> packagePath
plan symlink targetPath -> packagePath
```

Mutation order:

```text
create package parent directories
move target file into package
create target symlink pointing back to package file
```

Capture is dry-run by default and requires `--yes` to mutate, because it moves real files.

### `release <target-link>`

`release` stops managing one linked file and materializes the real file back at the target path.

Input:

```text
context
target-link
```

Algorithm:

```text
targetPath = expand target-link without following final symlink
classification = classify targetPath in context

reject unless classification is ManagedSymlink
owner = classification.owner

packagePath = join(owner.packageRoot, owner.packageRel)
expectedTarget = join(targetRoot(context), owner.packageRel)

reject if targetPath != expectedTarget
reject if packagePath does not exist
reject if packagePath is a directory

plan remove symlink targetPath
plan move packagePath -> targetPath
```

Mutation order:

```text
remove target symlink
move package file to the target path
```

After release, the package no longer contains that file. Empty package directories are left in place unless a future prune command removes them explicitly.

## Listing algorithms

### `packages`

Algorithm:

```text
for each enabled source in configured display order:
    base = packageBase(source, context)
    list direct child directories as packages
    display grouped by source
```

If the same package name appears in multiple sources, display both with source-qualified names.

### `tree [package]`

Without a package:

```text
show package tree grouped by source and package
```

With a package:

```text
resolve package for existing-package operation
show packageRoot tree
```

## Execution planning

The CLI should represent mutations as explicit actions:

```text
Mkdir(path)
Symlink(linkPath, payload)
RemoveSymlink(path)
Move(src, dst)
```

Planning rules:

1. Resolve all sources, packages, target paths, and ownership before mutating.
2. Accumulate all conflicts.
3. If any conflict exists, print them and exit non-zero.
4. If no conflicts exist, print the planned actions.
5. Mutate only when the command is allowed to execute after applying execution mode options.

Recommended defaults:

- `link`, `unlink`, and `relink` execute by default once the plan is conflict-free. `--dry-run` suppresses mutations.
- `capture` and `release` are dry-run by default and require `--yes` to mutate, because they move real files between the target tree and package tree.
- Root context mutations should make privilege requirements visible in the plan before attempting escalation.

## Validation examples

### User link

Input:

```text
tuck link zsh
```

Resolution:

```text
context = user
package ref = zsh
source candidates = sources with <source.path>/zsh
target root = $HOME
```

If only `public:zsh` exists, link package leaves into `$HOME`.

### Ambiguous public/private package

Input:

```text
tuck link ssh
```

If both `public:ssh` and `private:ssh` exist:

```text
error: ambiguous package "ssh"
hint: use public:ssh or private:ssh
```

### Explicit private package

Input:

```text
tuck link private:ssh
```

Resolution:

```text
source = private
context = user
package root = ~/.dotfiles-private/ssh
target root = $HOME
```

### Root package

Input:

```text
tuck root link sshd
```

Resolution:

```text
context = root
package root = ~/.dotfiles/.root/sshd
target root = /
```

Package file:

```text
~/.dotfiles/.root/sshd/etc/ssh/sshd_config
```

Target link:

```text
/etc/ssh/sshd_config
```

### Capture existing user file

Input:

```text
tuck capture private:ssh ~/.ssh/config
```

Plan:

```text
mkdir ~/.dotfiles-private/ssh/.ssh
move ~/.ssh/config -> ~/.dotfiles-private/ssh/.ssh/config
symlink ~/.ssh/config -> ~/.dotfiles-private/ssh/.ssh/config
```

### Release managed file

Input:

```text
tuck release ~/.ssh/config
```

If `~/.ssh/config` is a managed symlink to `~/.dotfiles-private/ssh/.ssh/config`, plan:

```text
remove symlink ~/.ssh/config
move ~/.dotfiles-private/ssh/.ssh/config -> ~/.ssh/config
```

### Link conflict

Input:

```text
tuck link git
```

If package file `~/.dotfiles/git/.gitconfig` maps to target `~/.gitconfig` and `~/.gitconfig` already exists as a real file:

```text
error: target exists as real file
hint: use capture if you want to move it into a package
```

### Managed by another package

If `~/.gitconfig` points to `~/.dotfiles-private/work/.gitconfig`, then `tuck link public:git` must fail:

```text
error: ~/.gitconfig is already managed by private:user:work
```
