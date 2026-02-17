#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Building WASM from $SCRIPT_DIR"

cd "$SCRIPT_DIR"
GOOS=js GOARCH=wasm go build -o ../server/static/main.wasm .

echo "✓ WASM compiled to ../server/static/main.wasm"
