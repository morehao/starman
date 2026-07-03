#!/bin/sh
set -e

REPO="morehao/starman"
BIN="starman"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

info()  { printf "${GREEN}%s${NC}\n" "$*"; }
error() { printf "${RED}%s${NC}\n" "$*" >&2; exit 1; }

FORCE=0
for arg in "$@"; do
  case "$arg" in
    --force) FORCE=1 ;;
    --help|-h) printf "Usage: %s [--force]\n  --force  Force reinstall even if already installed\n" "$0"; exit 0 ;;
  esac
done

detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$OS" in
    linux|darwin) ;;
    *) error "Unsupported OS: $OS" ;;
  esac

  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH" ;;
  esac
}

get_latest_version() {
  VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

  if [ -z "$VERSION" ]; then
    error "Failed to get latest version"
  fi
}

install() {
  INSTALL_DIR="/usr/local/bin"
  if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi

  if [ "$FORCE" -eq 0 ] && [ -x "$INSTALL_DIR/$BIN" ]; then
    INSTALLED_VER=$("$INSTALL_DIR/$BIN" --version 2>/dev/null | head -n1)
    if [ "$INSTALLED_VER" = "$VERSION" ]; then
      info "starman ${VERSION} is already installed at ${INSTALL_DIR}/${BIN}"
      info "Use --force to reinstall"
      exit 0
    fi
  fi

  TARBALL="${BIN}_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

  TMPDIR=$(mktemp -d)
  trap 'rm -rf "$TMPDIR"' EXIT

  info "Downloading ${TARBALL}..."
  curl -sL "$URL" -o "$TMPDIR/$TARBALL" || error "Download failed"

  tar xzf "$TMPDIR/$TARBALL" -C "$TMPDIR"

  mv "$TMPDIR/$BIN" "$INSTALL_DIR/$BIN"
  chmod +x "$INSTALL_DIR/$BIN"

  info "starman ${VERSION} installed to ${INSTALL_DIR}/${BIN}"

  if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    printf "Add %s to your PATH:\n  export PATH=\"%s:\$PATH\"\n" "$INSTALL_DIR" "$INSTALL_DIR"
  fi
}

detect_platform
get_latest_version
install
