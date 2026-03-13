#!/bin/bash
# Seven Test TUI Installer
# Usage: curl -sSL https://raw.githubusercontent.com/tbilbe/test-tui/main/install.sh | bash

set -e

REPO="tbilbe/test-tui"
INSTALL_DIR="${HOME}/.local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="x86_64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# macOS universal binary is just "darwin"
if [ "$OS" = "darwin" ]; then
  ARCH="universal"
fi

echo "Detecting system: ${OS}_${ARCH}"

# Get latest release tag
LATEST=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
echo "Latest version: ${LATEST}"

# Build download URL
TARBALL="seven-test-tui_${LATEST#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}"

echo "Downloading ${URL}..."

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download and extract
curl -sSL "$URL" | tar -xz -C "$INSTALL_DIR" seven-test-tui

# Remove quarantine on macOS
if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "$INSTALL_DIR/seven-test-tui" 2>/dev/null || true
fi

echo ""
echo "✅ Installed to ${INSTALL_DIR}/seven-test-tui"
echo ""

# Check if install dir is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "Add to your PATH by running:"
  echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
  echo "  source ~/.zshrc"
  echo ""
fi

echo "Run with: seven-test-tui"
