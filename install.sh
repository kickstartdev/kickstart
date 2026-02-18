#!/bin/sh
set -e

OWNER="kickstartdev"
REPO="kickstart"
BINARY="kickstartsh"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

LATEST=$(curl -sSf "https://api.github.com/repos/${OWNER}/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)

if [ -z "$LATEST" ]; then
    echo "Failed to fetch latest release"
    exit 1
fi

NAME="${BINARY}_${LATEST#v}_${OS}_${ARCH}"
URL="https://github.com/${OWNER}/${REPO}/releases/download/${LATEST}/${NAME}.tar.gz"

echo "Downloading ${BINARY} ${LATEST} for ${OS}/${ARCH}..."

TMP=$(mktemp -d)
curl -sSfL "$URL" -o "$TMP/release.tar.gz"
tar -xzf "$TMP/release.tar.gz" -C "$TMP"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
sudo install "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
rm -rf "$TMP"

echo "${BINARY} ${LATEST} installed successfully!"
