# tuck

[![CI](https://github.com/trippwill/tuck/actions/workflows/ci.yml/badge.svg)](https://github.com/trippwill/tuck/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/trippwill/tuck?sort=semver)](https://github.com/trippwill/tuck/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/trippwill/tuck.svg)](https://pkg.go.dev/github.com/trippwill/tuck)
[![License](https://img.shields.io/github/license/trippwill/tuck)](LICENSE)

`tuck` manages dotfiles as packages. It keeps a source repository of package
files, then deploys package leaves into your home directory or root filesystem.
Mutating target-tree commands print a plan by default and only change files when
you pass `--apply`. It is inspired by [GNU Stow](https://www.gnu.org/software/stow/)
and its package-directory model, with tuck-specific planning, status, copy-mode,
and root-context behavior.

## Install

If you already have Go 1.26 or newer:

```sh
go install github.com/trippwill/tuck/cmd/tuck@latest
```

For released binaries:

```sh
curl -fsSLo install.sh https://raw.githubusercontent.com/trippwill/tuck/main/install.sh
less install.sh
sh install.sh
```

Set `TUCK_VERSION=vX.Y.Z` to install a specific release, or
`INSTALL_DIR=/path/to/bin` to override the default `~/.local/bin`.

## Quickstart

Create or register a dotfiles source:

```sh
tuck source add ~/dotfiles --init --default
```

Adopt an existing file into a package:

```sh
tuck adopt ~/.zshrc zsh
tuck adopt ~/.zshrc zsh --apply
```

Deploy a package:

```sh
tuck package use zsh
tuck package use zsh --apply
```

Check what tuck manages:

```sh
tuck status ~/.zshrc
tuck package status zsh
```

## Common commands

| Command | What it does |
| --- | --- |
| `tuck source add <path> --init --default` | Create/register a dotfiles source and make it the default. |
| `tuck source list` | Show enabled sources on this machine. |
| `tuck package list` | List packages in the active source. |
| `tuck package show <package>` | Show a package's file tree. |
| `tuck package use <package>` | Plan package deployment. Add `--apply` to execute. |
| `tuck package drop <package>` | Plan removing managed deployments. Add `--apply` to execute. |
| `tuck package refresh <package>` | Plan drop-then-use refresh. Add `--apply` to execute. |
| `tuck adopt <file> <package>` | Plan moving a real target file into a package. Add `--apply` to execute. |
| `tuck eject <file>` | Plan moving a managed file back to the target tree. Add `--apply` to execute. |
| `tuck status <file>` | Classify one target path. |

Use `tuck package` as `tuck pkg` if you prefer shorter commands.

## Safety notes

- `adopt`, `eject`, `package use`, `package drop`, and `package refresh` are
  dry-run by default. Re-run with `--apply` after reviewing the plan.
- `--root` selects the root context for paths under `/`; tuck never escalates
  privileges for you.
- `--json` emits one machine-readable envelope for commands that produce tuck
  output.
- Per-file package config can use `deploy = "copy"` for apps that do not tolerate
  symlinks.

## Shell completion

Generate shell completion scripts with:

```sh
tuck completion bash
tuck completion zsh
tuck completion fish
```

For example, Bash users can source it directly:

```sh
source <(tuck completion bash)
```

## Documentation

- [`docs/cli-spec.md`](docs/cli-spec.md) is the detailed command and behavior
  specification.
- [`docs/testing-strategy.md`](docs/testing-strategy.md) explains how behavior is
  tested.
