#!/usr/bin/env bash
# install-hooks.sh — Install Git hooks for this repository.
# Run once after cloning: bash scripts/install-hooks.sh

set -euo pipefail

HOOKS_DIR="$(git rev-parse --git-dir)/hooks"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

install_hook() {
  local name="$1"
  local src="$SCRIPT_DIR/hooks/$name"
  local dst="$HOOKS_DIR/$name"

  if [ ! -f "$src" ]; then
    echo "  SKIP  $name (source not found)"
    return
  fi

  cp "$src" "$dst"
  chmod +x "$dst"
  echo "  INSTALL $dst"
}

install_hook "pre-commit"

echo ""
echo "✓ Git hooks installed."
