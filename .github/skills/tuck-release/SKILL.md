---
name: tuck-release
description: Release workflow for the tuck repository. Use when preparing, cutting, publishing, or verifying a tuck release; updating CHANGELOG.md for a release; creating or pushing v* tags; checking GitHub Actions release assets and checksums; verifying install.sh against a release; or troubleshooting the repo's CI/release/Pages release process.
---

# tuck release

Use this skill to release `github.com/trippwill/tuck` without rediscovering the
repo-specific process.

## Invariants

- Release tags are `v*` and `--version` is stamped with the exact tag.
- `main` is protected by the CI `check` status; use PRs unless the maintainer
  explicitly chooses otherwise.
- Local gate: `mise run check`.
- Coverage reporting: `mise run coverage`; informational only, no threshold.
- Release workflow: `.github/workflows/release.yml`.
- Release artifacts: linux/darwin amd64/arm64 `.tar.gz` archives plus
  `checksums.txt`.
- Release publishing uses GitHub generated notes.
- `install.sh` downloads release assets and verifies checksums.
- Docs Pages deploy from `main`, not tags.

## Changelog workflow

Before tagging:

1. Read `CHANGELOG.md`.
2. Move meaningful `Unreleased` bullets into a new `## vX.Y.Z` section.
3. Leave a fresh `## Unreleased` section above it.
4. Keep bullets user-facing. Do not list every commit.
5. Commit changelog changes before tagging.

## Preflight

1. Confirm the worktree is clean:

   ```sh
   git --no-pager status --short
   ```

2. Confirm `main` is up to date:

   ```sh
   git fetch origin
   git --no-pager status -sb
   ```

3. Run the local gate:

   ```sh
   mise run check
   ```

4. Optionally print coverage:

   ```sh
   mise run coverage
   ```

5. Confirm CI is green on `main`:

   ```sh
   gh run list --workflow CI --branch main --limit 3
   ```

## Cut a release

1. Choose the tag, for example `v0.1.0`.
2. Update and commit `CHANGELOG.md`.
3. Create and push the tag:

   ```sh
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

4. Watch the release workflow:

   ```sh
   gh run list --workflow Release --limit 1
   gh run watch <run-id>
   ```

## Verify the release

After the workflow succeeds:

1. Confirm the release exists:

   ```sh
   gh release view vX.Y.Z
   ```

2. Confirm expected assets:

   ```sh
   gh release view vX.Y.Z --json assets --jq '.assets[].name'
   ```

   Expected names include:

   - `tuck_vX.Y.Z_linux_amd64.tar.gz`
   - `tuck_vX.Y.Z_linux_arm64.tar.gz`
   - `tuck_vX.Y.Z_darwin_amd64.tar.gz`
   - `tuck_vX.Y.Z_darwin_arm64.tar.gz`
   - `checksums.txt`

3. Test the installer with the tag:

   ```sh
   tmp="$(mktemp -d)"
   INSTALL_DIR="$tmp/bin" TUCK_VERSION=vX.Y.Z sh ./install.sh
   "$tmp/bin/tuck" --version
   rm -rf "$tmp"
   ```

4. Confirm Pages is still configured from Actions:

   ```sh
   gh api repos/trippwill/tuck/pages --jq '{build_type, html_url}'
   ```

## Failure notes

- If the release workflow fails before publishing, fix the issue, delete the tag
  locally/remotely only if the maintainer approves, then re-tag.
- If a GitHub Release was created with bad assets, prefer deleting the failed
  release and tag before recreating it. Do not overwrite release assets silently.
- If `install.sh` fails checksum verification, treat the release as bad until the
  asset/checksum mismatch is understood.
