#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$SCRIPT_DIR/bin"
OUTPUT="$BIN_DIR/clip-tool"
LINK_PATH="$ROOT_DIR/clip-tool"
LINK_TARGET="clip-tool-src/bin/clip-tool"

mkdir -p "$BIN_DIR"

cd "$SCRIPT_DIR"
go build -o "$OUTPUT" ./cmd/clip-tool

rm -f "$LINK_PATH"
ln -s "$LINK_TARGET" "$LINK_PATH"

printf 'Built %s\n' "$OUTPUT"
printf 'Linked %s -> %s\n' "$LINK_PATH" "$LINK_TARGET"
