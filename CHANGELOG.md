# Changelog

All notable changes to tuck are tracked here.

## Unreleased

- Render `package show` output as an ASCII tree for easier package review (#6).
- Label `deploy = "copy"` entries in `package show` output (#12).
- Add an explicit source id override when registering sources with `source add` (#7).
- Add the adopt-on-conflict shortcut for `package use` (#8).
- Flip `adopt` arguments to package-first (`adopt <package> <file>`) (#9).
- Harden beta behavior coverage for help/version JSON handling, source warnings,
  state repair hints, empty sources, and `--no-color` precedence (#1-#5).

## v0.1.0

- Initial release of tuck's plan-first dotfile package workflow.
- Add source registration, package listing/show/status, package use/drop/refresh,
  adopt/eject, home/root contexts, and shell completion.
- Add symlink and copy deployment modes with copied-file drift tracking.
- Add stable human and JSON output, error codes, and acceptance coverage.
- Add install script, release builds, checksums, CI, documentation website, and
  contributor/release docs.
