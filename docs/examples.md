---
title: Examples
---

# Worked examples

## Create a source

```sh
mkdir -p ~/dotfiles
tuck source add ~/dotfiles --init --default
tuck source list
```

## Adopt a shell config

```sh
tuck adopt ~/.zshrc zsh
tuck adopt ~/.zshrc zsh --apply
tuck package show zsh
```

## Deploy a package on another machine

```sh
tuck source add ~/dotfiles --default
tuck package list
tuck package use zsh
tuck package use zsh --apply
```

## Manage a root-context file

Root-context package files live under `.root` inside the source and deploy under
`/`:

```sh
tuck adopt --root /etc/ssh/sshd_config sshd
sudo tuck adopt --root /etc/ssh/sshd_config sshd --apply
tuck package use --root sshd
sudo tuck package use --root sshd --apply
```

## Use copy deployment for symlink-hostile apps

Configure one package leaf to deploy as a real copied file:

```sh
tuck package config set app .config/app/config --deploy copy
tuck package use app
tuck package use app --apply
```

Then inspect drift:

```sh
tuck status ~/.config/app/config
tuck package status app
```

## Eject a managed file

```sh
tuck eject ~/.zshrc
tuck eject ~/.zshrc --apply
```

Eject stops managing the target and leaves a real file in place.
