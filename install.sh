#!/bin/sh
set -eu

repo="trippwill/tuck"
install_dir="${INSTALL_DIR:-"$HOME/.local/bin"}"
version="${TUCK_VERSION:-}"

case "${1:-}" in
	-h|--help)
		cat <<'USAGE'
Usage: install.sh

Environment:
  TUCK_VERSION=vX.Y.Z   install a specific release instead of latest
  INSTALL_DIR=PATH      install directory, defaults to ~/.local/bin
USAGE
		exit 0
		;;
esac

need() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "error: $1 is required" >&2
		exit 1
	fi
}

need curl
need install
need tar

if [ -z "$version" ]; then
	latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")"
	version="${latest_url##*/}"
fi

case "$version" in
	v*) ;;
	*)
		echo "error: could not determine release version" >&2
		exit 1
		;;
esac

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
	linux|darwin) ;;
	*)
		echo "error: unsupported OS: $os" >&2
		exit 1
		;;
esac

case "$(uname -m)" in
	x86_64|amd64) arch="amd64" ;;
	arm64|aarch64) arch="arm64" ;;
	*)
		echo "error: unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

name="tuck_${version}_${os}_${arch}"
asset="${name}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${version}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

curl -fsSL --retry 3 -o "$tmp/$asset" "$base_url/$asset"
curl -fsSL --retry 3 -o "$tmp/checksums.txt" "$base_url/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
	(cd "$tmp" && awk -v asset="$asset" '$2 == asset { print }' checksums.txt | sha256sum -c - >/dev/null)
elif command -v shasum >/dev/null 2>&1; then
	want="$(awk -v asset="$asset" '$2 == asset { print $1 }' "$tmp/checksums.txt")"
	got="$(shasum -a 256 "$tmp/$asset" | awk '{ print $1 }')"
	if [ -z "$want" ] || [ "$want" != "$got" ]; then
		echo "error: checksum mismatch for $asset" >&2
		exit 1
	fi
else
	echo "error: sha256sum or shasum is required" >&2
	exit 1
fi

tar -xzf "$tmp/$asset" -C "$tmp"
mkdir -p "$install_dir"
install -m 0755 "$tmp/$name/tuck" "$install_dir/tuck"

echo "installed tuck to $install_dir/tuck"
