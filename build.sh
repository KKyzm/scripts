#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/bin"

mkdir -p "$BIN_DIR"

# Build clip-tool
echo "Building clip-tool..."
"$SCRIPT_DIR/clip-tool-src/build.sh"

# Create symlinks for all tools
echo "Creating symlinks..."
ln -sf ../clip-tool-src/bin/clip-tool "$BIN_DIR/clip-tool"
ln -sf ../remote-src/remote "$BIN_DIR/remote"

echo ""
echo "Done. Tools available in $BIN_DIR/:"
ls -la "$BIN_DIR"
