# tuck

Dotfile package manager.

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

Set `TUCK_VERSION=vX.Y.Z` to install a specific release, or `INSTALL_DIR=/path/to/bin` to override the default `~/.local/bin`.
