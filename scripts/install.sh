#!/bin/sh
# Install a wicked-sqlc release binary after checking its SHA-256.
#
#   scripts/install.sh            # latest release
#   scripts/install.sh v2.4.1     # a given release
#   curl -fsSL https://raw.githubusercontent.com/Stumble/sqlc/main/scripts/install.sh | sh -s -- v2.4.1
#
# Environment:
#   INSTALL_DIR   where to put sqlc (default: $(go env GOPATH)/bin, else ~/.local/bin)
#   SQLC_SHA256   expected SHA-256 of the archive; pin it to trust only that
#                 archive. Without it the release's checksums.txt is used.
#
# Linux (amd64, arm64; static binary) and macOS (amd64, arm64, 12+). On
# Windows download sqlc_<version>_windows_amd64.zip from the release.
set -eu

REPO="${SQLC_REPO:-Stumble/sqlc}"
BASE_URL="${SQLC_BASE_URL:-https://github.com/$REPO/releases}"
VERSION="${1:-${SQLC_VERSION:-}}"

die() { echo "install.sh: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required"; }
need curl
need tar

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) die "unsupported OS $(uname -s); download the archive from $BASE_URL" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) die "unsupported architecture $(uname -m)" ;;
esac

if [ -z "$VERSION" ]; then
  # .../releases/latest redirects to .../releases/tag/<tag>
  VERSION="$(curl -fsSLo /dev/null -w '%{url_effective}' "$BASE_URL/latest")"
  VERSION="${VERSION##*/}"
fi
case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac

if [ -z "${INSTALL_DIR:-}" ]; then
  if command -v go >/dev/null 2>&1; then
    INSTALL_DIR="$(go env GOPATH)/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi

archive="sqlc_${VERSION#v}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $archive ($VERSION)"
curl -fsSL -o "$tmp/$archive" "$BASE_URL/download/$VERSION/$archive"

if [ -n "${SQLC_SHA256:-}" ]; then
  want="$SQLC_SHA256"
else
  curl -fsSL -o "$tmp/checksums.txt" "$BASE_URL/download/$VERSION/checksums.txt"
  want="$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")"
  [ -n "$want" ] || die "$archive is not in checksums.txt"
fi
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "$tmp/$archive" | awk '{ print $1 }')"
else
  need shasum
  got="$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')"
fi
[ "$got" = "$want" ] || die "SHA-256 mismatch for $archive: got $got, want $want"

tar -xzf "$tmp/$archive" -C "$tmp" sqlc
mkdir -p "$INSTALL_DIR"
mv "$tmp/sqlc" "$INSTALL_DIR/sqlc"
chmod 0755 "$INSTALL_DIR/sqlc"
echo "Installed $INSTALL_DIR/sqlc: $("$INSTALL_DIR/sqlc" version)"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Note: $INSTALL_DIR is not on PATH" ;;
esac
