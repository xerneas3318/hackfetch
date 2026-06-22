#!/bin/sh
# hackfetch installer
# usage:   curl -fsSL https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.sh | sh
# override install dir:
#   HACKFETCH_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.sh | sh
#
# POSIX sh compatible. Auto-installs prereqs (curl, tar) on Linux via the
# system package manager when possible. Works on Linux (glibc + musl) and macOS.

set -eu

REPO="xerneas3318/hackfetch"
INSTALL_DIR="${HACKFETCH_INSTALL_DIR:-}"

# --- colors (only when stdout is a tty)
if [ -t 1 ]; then
  bold=$(printf '\033[1m')
  green=$(printf '\033[0;32m')
  orange=$(printf '\033[0;38;5;208m')
  red=$(printf '\033[0;31m')
  dim=$(printf '\033[2m')
  reset=$(printf '\033[0m')
else
  bold=""; green=""; orange=""; red=""; dim=""; reset=""
fi

say() { printf '%s\n' "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- pick a sudo prefix when we need one
sudo_prefix() {
  if [ "$(id -u)" = "0" ]; then
    printf ''
  elif have sudo; then
    printf 'sudo'
  else
    printf ''
  fi
}

# --- detect linux package manager
detect_pm() {
  if have apt-get; then echo apt
  elif have dnf;     then echo dnf
  elif have yum;     then echo yum
  elif have pacman;  then echo pacman
  elif have zypper;  then echo zypper
  elif have apk;     then echo apk
  elif have brew;    then echo brew
  else echo ""
  fi
}

# --- install a list of packages via the detected pm
pm_install() {
  pm=$1; shift
  pkgs=$*
  SUDO=$(sudo_prefix)
  case "$pm" in
    apt)    $SUDO apt-get update -y >/dev/null && $SUDO apt-get install -y $pkgs ;;
    dnf)    $SUDO dnf install -y $pkgs ;;
    yum)    $SUDO yum install -y $pkgs ;;
    pacman) $SUDO pacman -Sy --noconfirm --needed $pkgs ;;
    zypper) $SUDO zypper --non-interactive install $pkgs ;;
    apk)    $SUDO apk add --no-cache $pkgs ;;
    brew)   brew install $pkgs ;;
    *)      return 1 ;;
  esac
}

# --- ensure a tool is present; try to install it if missing
ensure_tool() {
  tool=$1
  pkg=${2:-$1}
  if have "$tool"; then return 0; fi
  pm=$(detect_pm)
  if [ -z "$pm" ]; then
    say "  ${red}error:${reset} $tool is required but no supported package manager was found"
    say "  ${dim}install $tool manually and re-run${reset}"
    return 1
  fi
  say "  ${dim}+ installing missing prereq: $tool (via $pm)${reset}"
  if ! pm_install "$pm" "$pkg" >/dev/null 2>&1; then
    say "  ${red}error:${reset} failed to install $tool via $pm"
    say "  ${dim}install $tool manually and re-run${reset}"
    return 1
  fi
}

# --- header
say ""
say "  ${orange}✦ hackfetch installer${reset}"
say ""

# --- prereqs: curl + tar are required. on Linux setup, xdg-open is nice for -setup.
os_name=$(uname -s)
case "$os_name" in
  Linux)
    ensure_tool curl curl
    ensure_tool tar  tar
    # xdg-open is optional but used by `hackfetch -setup` to open the auth page.
    if ! have xdg-open; then
      pm=$(detect_pm)
      if [ -n "$pm" ]; then
        say "  ${dim}+ installing optional: xdg-utils (for hackfetch -setup)${reset}"
        case "$pm" in
          apk) pm_install "$pm" xdg-utils >/dev/null 2>&1 || true ;;
          *)   pm_install "$pm" xdg-utils >/dev/null 2>&1 || true ;;
        esac
      fi
    fi
    ;;
  Darwin)
    # macOS ships curl + tar. nothing to install.
    if ! have curl || ! have tar; then
      say "  ${red}error:${reset} curl and tar are required (and missing). install Xcode CLT: xcode-select --install"
      exit 1
    fi
    ;;
  *)
    say "  ${red}error:${reset} unsupported os: $os_name"
    exit 1
    ;;
esac

# --- detect platform
case "$os_name" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) say "  ${red}error:${reset} unsupported arch: $(uname -m)"; exit 1 ;;
esac
platform="$os-$arch"

say "  ${green}✓${reset} detected: ${bold}$platform${reset}"

# --- pick install dir
SUDO=""
if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ]; then
    INSTALL_DIR="/usr/local/bin"
  elif [ "$(id -u)" = "0" ]; then
    INSTALL_DIR="/usr/local/bin"
  elif have sudo; then
    INSTALL_DIR="/usr/local/bin"
    SUDO="sudo"
  else
    mkdir -p "$HOME/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi

# --- latest release tag
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
trap 'rm -rf "$tmpdir"' EXIT INT TERM HUP

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
