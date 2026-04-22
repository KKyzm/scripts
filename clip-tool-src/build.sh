#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/bin"
OUTPUT="$BIN_DIR/clip-tool"

mkdir -p "$BIN_DIR"

cd "$SCRIPT_DIR"
go build -o "$OUTPUT" ./cmd/clip-tool

printf 'Built %s\n' "$OUTPUT"
