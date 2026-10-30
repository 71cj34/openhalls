#!/bin/sh

set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
go build -o "$SCRIPT_DIR/../../hallview" "$SCRIPT_DIR"
exec "$SCRIPT_DIR/../../hallview" "$@"
