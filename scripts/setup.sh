#!/usr/bin/env bash
# setup.sh — Interactive project setup for go-api-skeletton.
#
# Usage:
#   ./scripts/setup.sh
#   make setup
#
# What it does:
#   1. Checks required tools (go, docker, make)
#   2. Asks for the new Go module name and renames it everywhere
#   3. Creates .env from .env.example with mandatory variables filled in

set -euo pipefail

# ─── Colours ────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${CYAN}${BOLD}→${RESET} $*"; }
success() { echo -e "${GREEN}${BOLD}✓${RESET} $*"; }
warn()    { echo -e "${YELLOW}${BOLD}!${RESET} $*"; }
error()   { echo -e "${RED}${BOLD}✗${RESET} $*" >&2; }
die()     { error "$*"; exit 1; }

# ─── Resolve script location (works even when called from another dir) ───────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# ─── 1. Check required tools ────────────────────────────────────────────────
echo
echo -e "${BOLD}Checking required tools…${RESET}"

check_tool() {
  local cmd="$1" install_hint="$2"
  if command -v "$cmd" &>/dev/null; then
    success "$cmd $(${cmd} version 2>/dev/null | head -1 | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)"
  else
    die "$cmd not found. $install_hint"
  fi
}

check_tool go    "Install from https://go.dev/dl/"
check_tool make  "Install via your package manager (apt install make / brew install make)"

# Docker: check daemon is reachable, not just the CLI
if command -v docker &>/dev/null; then
  if docker info &>/dev/null 2>&1; then
    success "docker $(docker version --format '{{.Client.Version}}' 2>/dev/null)"
  else
    die "Docker CLI found but the daemon is not running. Start Docker and retry."
  fi
else
  die "docker not found. Install from https://docs.docker.com/get-docker/"
fi

# ─── 2. Rename Go module ────────────────────────────────────────────────────
echo
echo -e "${BOLD}Go module rename${RESET}"

CURRENT_MODULE=$(grep '^module ' "$ROOT_DIR/go.mod" | awk '{print $2}')
info "Current module: ${BOLD}${CURRENT_MODULE}${RESET}"

while true; do
  echo -en "${CYAN}${BOLD}?${RESET} New module path (leave blank to keep current): "
  read -r NEW_MODULE
  NEW_MODULE="${NEW_MODULE:-$CURRENT_MODULE}"

  # Basic validation: must look like a Go module path
  if [[ "$NEW_MODULE" =~ ^[a-zA-Z0-9._/-]+$ ]]; then
    break
  else
    warn "Invalid module path — use only letters, digits, '.', '_', '-', '/'."
  fi
done

if [[ "$NEW_MODULE" != "$CURRENT_MODULE" ]]; then
  info "Renaming ${CURRENT_MODULE} → ${NEW_MODULE}…"

  # go.mod
  sed -i "s|^module ${CURRENT_MODULE}$|module ${NEW_MODULE}|" "$ROOT_DIR/go.mod"

  # All .go files
  find "$ROOT_DIR" -type f -name '*.go' \
    -not -path '*/vendor/*' \
    -exec sed -i "s|\"${CURRENT_MODULE}|\"${NEW_MODULE}|g" {} +

  # go.sum is regenerated, not renamed — just run tidy
  (cd "$ROOT_DIR" && go mod tidy 2>/dev/null)

  success "Module renamed to ${NEW_MODULE}"
else
  info "Module path unchanged."
fi

# ─── 3. Create .env ─────────────────────────────────────────────────────────
echo
echo -e "${BOLD}.env configuration${RESET}"

ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE="$ROOT_DIR/.env.example"

[[ -f "$ENV_EXAMPLE" ]] || die ".env.example not found at $ROOT_DIR"

if [[ -f "$ENV_FILE" ]]; then
  echo -en "${YELLOW}${BOLD}!${RESET} .env already exists. Overwrite? [y/N] "
  read -r overwrite
  if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
    info "Keeping existing .env — skipping."
    echo
    echo -e "${GREEN}${BOLD}Setup complete.${RESET} Run ${BOLD}make dev${RESET} to start."
    exit 0
  fi
fi

cp "$ENV_EXAMPLE" "$ENV_FILE"

# ── Helper: update a key in .env ────────────────────────────────────────────
set_env() {
  local key="$1" value="$2"
  # Escape special characters for sed replacement
  local escaped_value
  escaped_value=$(printf '%s\n' "$value" | sed 's/[[\.*^$()+?{|]/\\&/g')
  sed -i "s|^${key}=.*|${key}=${escaped_value}|" "$ENV_FILE"
}

# ── JWT_SECRET ───────────────────────────────────────────────────────────────
echo
echo -en "${CYAN}${BOLD}?${RESET} JWT_SECRET — auto-generate a 50-char secret? [Y/n] "
read -r gen_jwt

if [[ ! "$gen_jwt" =~ ^[Nn]$ ]]; then
  # Generate 50 printable chars (no special shell chars to avoid sed issues)
  JWT_SECRET=$(set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 50)
  set_env "JWT_SECRET" "$JWT_SECRET"
  success "JWT_SECRET auto-generated (50 chars, alphanumeric)"
else
  while true; do
    echo -en "${CYAN}${BOLD}?${RESET} JWT_SECRET (min 32 chars): "
    read -r JWT_SECRET
    if [[ ${#JWT_SECRET} -ge 32 ]]; then
      set_env "JWT_SECRET" "$JWT_SECRET"
      success "JWT_SECRET set"
      break
    else
      warn "Too short — minimum 32 characters."
    fi
  done
fi

# ── Admin credentials ────────────────────────────────────────────────────────
echo
info "Admin seed account (used when SEED_ADMIN=true)"

while true; do
  echo -en "${CYAN}${BOLD}?${RESET} ADMIN_EMAIL: "
  read -r ADMIN_EMAIL
  if [[ "$ADMIN_EMAIL" =~ ^[^@]+@[^@]+\.[^@]+$ ]]; then
    set_env "ADMIN_EMAIL" "$ADMIN_EMAIL"
    break
  else
    warn "Invalid email address."
  fi
done

while true; do
  echo -en "${CYAN}${BOLD}?${RESET} ADMIN_PASSWORD (min 8 chars): "
  # Hide input
  read -rs ADMIN_PASSWORD
  echo
  if [[ ${#ADMIN_PASSWORD} -ge 8 ]]; then
    set_env "ADMIN_PASSWORD" "$ADMIN_PASSWORD"
    success "Admin credentials set"
    break
  else
    warn "Too short — minimum 8 characters."
  fi
done

# ── Enable seed? ─────────────────────────────────────────────────────────────
echo -en "${CYAN}${BOLD}?${RESET} Enable SEED_ADMIN on first start? [y/N] "
read -r seed
if [[ "$seed" =~ ^[Yy]$ ]]; then
  set_env "SEED_ADMIN" "true"
  success "SEED_ADMIN=true"
fi

# ── APP_ENV ──────────────────────────────────────────────────────────────────
echo -en "${CYAN}${BOLD}?${RESET} APP_ENV [development/production] (default: development): "
read -r APP_ENV
APP_ENV="${APP_ENV:-development}"
if [[ "$APP_ENV" == "production" ]]; then
  set_env "APP_ENV" "production"
  success "APP_ENV=production"
else
  success "APP_ENV=development"
fi

# ─── Done ────────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}${BOLD}Setup complete.${RESET}"
echo
echo "  Next steps:"
echo -e "    ${BOLD}make dev${RESET}        — start the server with hot reload"
echo -e "    ${BOLD}make test${RESET}       — run unit tests"
echo -e "    ${BOLD}make scaffold NAME=<resource>${RESET}  — generate a new domain module"
echo
