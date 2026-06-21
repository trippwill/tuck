---
title: Quickstart
---

# Quickstart

## 1. Install tuck

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

## 2. Register a source

A source is the dotfiles repository tuck reads packages from:

```sh
tuck source add ~/dotfiles --init --default
```

This writes `~/dotfiles/.tuck.toml` if it does not exist and records the source
in machine-local state.

## 3. Adopt an existing file

Adopt plans the move first:

```sh
tuck adopt ~/.zshrc zsh
```

Apply after reviewing the plan:

```sh
tuck adopt ~/.zshrc zsh --apply
```

## 4. Deploy a package

Plan deployment:

```sh
tuck package use zsh
```

Apply it:

```sh
tuck package use zsh --apply
```

## 5. Check status

```sh
tuck status ~/.zshrc
tuck package status zsh
```

## Notes

- Mutating target-tree commands are dry-run by default.
- Use `--root` for root-context files under `/`.
- Use `--json` for machine-readable output.
- `package` can be shortened to `pkg`.
