#!/usr/bin/env bash
# hackfetch installer
# usage: curl -fsSL https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.sh | bash
# optional env: HACKFETCH_INSTALL_DIR=~/.local/bin to override install location

set -euo pipefail

REPO="xerneas3318/hackfetch"
INSTALL_DIR="${HACKFETCH_INSTALL_DIR:-}"

bold=$'\033[1m'
green=$'\033[0;32m'
orange=$'\033[0;38;5;208m'
red=$'\033[0;31m'
dim=$'\033[2m'
reset=$'\033[0m'

say() { printf '%s\n' "$*"; }

# --- requirements
for tool in curl tar; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    say "${red}error:${reset} $tool is required but not installed"
    exit 1
  fi
done

# --- detect platform
case "$(uname -s)" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) say "${red}error:${reset} unsupported os: $(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) say "${red}error:${reset} unsupported arch: $(uname -m)"; exit 1 ;;
esac
platform="$os-$arch"

# --- header
say ""
say "  ${orange}✦ hackfetch installer${reset}"
say ""
say "  ${green}✓${reset} detected: ${bold}$platform${reset}"

# --- pick install dir
SUDO=""
if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
  elif [ "$(id -u)" = "0" ]; then
    INSTALL_DIR="/usr/local/bin"
  elif command -v sudo >/dev/null 2>&1; then
    INSTALL_DIR="/usr/local/bin"
    SUDO="sudo"
  else
    mkdir -p "$HOME/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi

# --- latest release
say "  ${dim}→ checking latest release...${reset}"
latest=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | head -n1 \
  | cut -d'"' -f4 || true)

if [ -z "$latest" ]; then
  say "  ${red}error:${reset} couldn't fetch latest release tag"
  exit 1
fi

say "  ${green}✓${reset} version: ${bold}$latest${reset}"

url="https://github.com/$REPO/releases/download/$latest/hackfetch-$platform.tar.gz"

# --- download + extract
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

say "  ${dim}↓ downloading $url${reset}"
if ! curl -fsSL "$url" -o "$tmpdir/hackfetch.tar.gz"; then
  say "  ${red}error:${reset} download failed: $url"
  say "  ${dim}check that the release has binaries for your platform${reset}"
  exit 1
fi

tar -xzf "$tmpdir/hackfetch.tar.gz" -C "$tmpdir"

# --- install
say "  ${dim}↓ installing to $INSTALL_DIR${reset}"
if [ -n "$SUDO" ]; then
  say "  ${dim}(sudo required to write to $INSTALL_DIR)${reset}"
fi
$SUDO install -m 0755 "$tmpdir/hackfetch" "$INSTALL_DIR/hackfetch"

say ""
say "  ${green}✓ installed${reset}  $INSTALL_DIR/hackfetch"
say ""

# --- PATH check
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    say "  ${orange}⚠ $INSTALL_DIR is not in your PATH${reset}"
    say "  ${dim}add this to your shell rc:${reset}"
    say ""
    say "    export PATH=\"$INSTALL_DIR:\$PATH\""
    say ""
    ;;
esac

say "  ${dim}next:${reset}  hackfetch -setup"
say ""
