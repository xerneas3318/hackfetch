# hackfetch

[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Hackatime](https://img.shields.io/badge/Hackatime-connected-ec3750?style=flat-square)](https://hackatime.hackclub.com)
[![Built for Stardance](https://img.shields.io/badge/Built%20for-Stardance-9b5cf6?style=flat-square)](https://stardance.hackclub.com)
[![Platforms](https://img.shields.io/badge/Platforms-Linux%20%7C%20macOS%20%7C%20Windows-2e7d32?style=flat-square)](#getting-started)
[![License PolyForm NC 1.0.0](https://img.shields.io/badge/License-PolyForm%20NC%201.0.0-blue?style=flat-square)](LICENSE)
[![Release v1.5.0](https://img.shields.io/badge/Release-v1.5.0-ec3750?style=flat-square)](https://github.com/xerneas3318/hackfetch/releases)

<p align="center">
  <img src="Images/stardance-ocean.png" alt="hackfetch stardance ocean" width="820">
</p>

**A Hack Club themed system fetch with live Hackatime stats.** Shows your system info next to a customizable Hack Club logo, plus your today/weekly hours, top project, top language, streak, and more. All from your terminal, in one keystroke.

hackfetch was built for [Stardance](https://stardance.hackclub.com), Hack Club's worldwide hackathon. It runs as a single Go binary with zero runtime dependencies, reads your existing `~/.wakatime.cfg`, and pulls live stats from [Hackatime](https://hackatime.hackclub.com) every time you run it.

## Contents

- [Why hackfetch](#why-hackfetch)
- [What it does](#what-it-does)
- [Gallery](#gallery)
- [Architecture](#architecture)
- [Getting started](#getting-started)
  - [1. Install](#1-install)
  - [2. Connect to Hackatime](#2-connect-to-hackatime)
- [Usage](#usage)
- [Logos and color schemes](#logos-and-color-schemes)
- [Custom themes](#custom-themes)
- [Live mode and card export](#live-mode-and-card-export)
- [Configuration](#configuration)
- [What you get from Hackatime](#what-you-get-from-hackatime)
- [Repository layout](#repository-layout)
- [Status](#status)
- [License](#license)

## Why hackfetch

Most "fetch" tools (neofetch, fastfetch, and the rest) show the same things: OS, kernel, CPU, memory, uptime. They are great, and also they do not know anything about what you are actually building.

Hack Club runs its own coding-time tracker, [Hackatime](https://hackatime.hackclub.com), a WakaTime-compatible backend that aggregates your real coding stats: today's hours, weekly total, top project, current streak. Those are the numbers worth caring about when you open a terminal.

hackfetch is what happens when you cross a fetch tool with that data and then style the whole thing in Hack Club colors and ASCII art.

- **Built for the Hack Club crowd.** Six built-in Hack Club logos, themed gradients, Pride flag palettes, and the Stardance countdown baked in.
- **Zero runtime dependencies.** Pure Go, single binary. Works on minimal Linux containers, fresh macOS installs, and stock Windows alike.
- **Live updating.** Open it with `-watch` and your stats refresh in place while you code.

## What it does

hackfetch runs as one Go binary that pulls four things together at once:

- **System fetch.** OS, hostname, user, shell, terminal, editor. The classic neofetch-style snapshot of the machine you are on, rendered next to a Hack Club logo.
- **Hackatime stats.** Today's coding time, 7-day total, current streak, top project, most-used language, top editor, top category. Pulled live from your Hackatime account every time you run the command.
- **Live mode.** `hackfetch -watch` keeps the fetch on screen and redraws every 30 seconds. Your hours tick up as you code, in the corner of your terminal.
- **Card export.** `hackfetch -export card.png` (or `.jpg`, or `.svg`) saves the current fetch as a shareable image with all colors preserved. Drop it into a devlog, a Slack channel, or a tweet.

All rendering happens locally in your terminal. No browser, no dashboard, no extra processes.

## Gallery

**Default Hack Club logo, forest gradient.**

<img src="Images/hackclub-forest.png" alt="hackclub forest" width="640">

**Trans flag colors on the default Hack Club logo.**

<img src="Images/hackclub-trans.png" alt="hackclub trans" width="640">

**The rocket logo in default colors.**

<img src="Images/rocket-hackclub.png" alt="rocket" width="640">

**The flag logo in sunset gradient.**

<img src="Images/flag-sunset.png" alt="flag sunset" width="640">

**The bot logo on a matrix-green scheme.**

<img src="Images/bot-matrix.png" alt="bot matrix" width="640">

**The Stardance logo in ocean gradient.**

<img src="Images/stardance-ocean.png" alt="stardance ocean" width="640">

## Architecture

```
   terminal ──► hackfetch (single Go binary)
                    │
                    ├──► local system info  (os, host, user, shell, terminal, editor)
                    │
                    ├──► Hackatime API      (today, weekly, streak, top project, top lang)
                    │        │
                    │        └──► smart fallbacks: when Hackatime says "unknown" for
                    │                              language, infer from heartbeat file
                    │                              extensions and label as (inferred)
                    │
                    ├──► layout engine      (pads each logo row, aligns info column)
                    │
                    └──► render targets     (ANSI 256-color terminal, or PNG/JPG/SVG card)
```

A single binary, one network round-trip per refresh, no daemon. The same render pipeline drives both the one-shot fetch and the `-watch` live mode. The card exporter shares the layout engine and can emit **PNG**, **JPEG**, or **SVG** depending on the output file extension; PNG/JPG are rasterized in-process using an embedded copy of DejaVu Sans Mono so no system fonts or external converters are needed.

## Getting started

### 1. Install

The fastest way on any Linux or macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.sh | sh
```

The installer auto-detects your OS, CPU architecture, and package manager (`apt`, `dnf`, `yum`, `pacman`, `zypper`, `apk`, or `brew`), installs any missing prereqs (`curl`, `tar`, `xdg-utils`), then drops the right binary into `/usr/local/bin` (or `~/.local/bin` if it can't sudo). POSIX `sh` compatible, so it also works on Alpine and minimal containers.

On Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.ps1 | iex
```

Installs `hackfetch.exe` to `%LOCALAPPDATA%\Programs\hackfetch` and adds that folder to your user `PATH`.

Alternative install paths:

```sh
# Homebrew (macOS, Linuxbrew)
brew tap xerneas3318/tap
brew install hackfetch

# From source with Go
go install github.com/xerneas3318/hackfetch@latest
```

### 2. Connect to Hackatime

hackfetch reads your API key from `~/.wakatime.cfg`. If you already use Hackatime or WakaTime, you're done.

If not, the easiest path is Hack Club's official setup:

```sh
curl -fsSL https://raw.githubusercontent.com/hackclub/hackatime-setup/main/install.sh | bash
```

Or run `hackfetch -setup` and it will walk you through the auth page at [hackatime.hackclub.com/my/wakatime_setup](https://hackatime.hackclub.com/my/wakatime_setup) and wait for the config file to land.

## Usage

```sh
hackfetch                              # defaults
hackfetch stardance rainbow            # positional shorthand: <logo> <color>
hackfetch logo flag color pride        # keyword form
hackfetch -logo orpheus -color ocean   # flag form
hackfetch -v                           # verbose: + top editor, top category
hackfetch -watch                       # live mode, refreshes every 30s
hackfetch -export card.png             # save the fetch as a shareable image (.png/.jpg/.svg)
hackfetch -list                        # show all logos and colors
hackfetch -h                           # help
hackfetch -setup                       # (re-)configure Hackatime
hackfetch -no-net                      # offline mode (skip API calls)
```

Flags go before positional args. `hackfetch -export card.png stardance pride` works; `hackfetch stardance pride -export card.png` does not.

## Logos and color schemes

| | Available |
|---|---|
| **Logos** | `hackclub`, `stardance`, `flag`, `orpheus`, `bot`, `rocket` |
| **Solid colors** | `hackclub`, `orange`, `mono`, `mute`, `matrix` |
| **Gradients** | `rainbow`, `sunset`, `ocean`, `forest`, `stardance` |
| **Pride flags** | `pride`, `trans`, `bi`, `pan` |
| **Special** | `auto` (defaults to `pride` in June, `hackclub` otherwise) |

Run `hackfetch -list` for the full set.

## Custom themes

Drop your own color schemes in `~/.config/hackfetch/colors.json`:

```json
{
  "schemes": {
    "vaporwave": {
      "colors": [199, 165, 99, 51],
      "mode": "per-line"
    },
    "fire": {
      "colors": [196, 202, 208, 214, 220, 226],
      "mode": "per-char"
    }
  }
}
```

`mode` is one of `single`, `per-line`, or `per-char`. `colors` are [ANSI 256 color codes](https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit). Your themes override built-ins of the same name. Then run:

```sh
hackfetch -color vaporwave
```

## Live mode and card export

**Live mode (`-watch`).** Re-fetches your Hackatime stats every 30 seconds and redraws in place. Today's hours tick up as you code. Ctrl+C to quit.

```sh
hackfetch -watch
hackfetch rocket -watch -color sunset
```

**Card export (`-export`).** Saves the current fetch as a shareable image with rounded corners, dark background, and every color preserved. The output format is picked from the file extension:

- `.png`: full-color raster, rendered in-process with an embedded monospace font. Best for uploading to Slack, Discord, Stardance devlogs, or anywhere else that won't accept SVG.
- `.jpg` / `.jpeg`: same rendering, JPEG-encoded at quality 92. Smaller file, no transparency.
- `.svg`: original vector output. Open it in any browser; scales cleanly to any size.

```sh
hackfetch -export card.png
hackfetch -export card.jpg stardance pride
hackfetch -export card.svg -logo orpheus -color rainbow
```

## Configuration

hackfetch reads these environment variables. Add them to your `~/.zshrc` or `~/.bashrc` to set defaults:

| Variable | Default | Purpose |
|---|---|---|
| `HACKFETCH_LOGO` | `hackclub` | Default logo (see `-list`). |
| `HACKFETCH_COLOR` | `hackclub` | Default color scheme. |
| `HACKFETCH_VERBOSE` | unset | Set to `1` to enable `-v` output by default. |
| `HACKFETCH_STARDUST` | unset | Your stardust count. Shown next to the ✦ field. |
| `HACKFETCH_INSTALL_DIR` | (auto) | Override the install directory used by `install.sh`. |
| `WAKATIME_HOME` | `$HOME` | Where to look for `.wakatime.cfg`. |

## What you get from Hackatime

When your `~/.wakatime.cfg` points at a working Hackatime account, hackfetch fetches and shows:

| Field | Meaning |
|---|---|
| **today** | Hours coded today. |
| **7-day total** | Hours coded over the past week. |
| **streak** | Consecutive days with activity. |
| **stardust ✦** | Your stardust count (set via `HACKFETCH_STARDUST=N`). |
| **stardance** | Days left until Stardance ends (auto-hides after Sep 30, 2026). |
| **slack** | Your Hack Club / Hackatime handle. |
| **top project** | Most-worked project (today and weekly). |
| **top language** | Most-used language. When Hackatime reports `unknown`, hackfetch infers from heartbeat file extensions and labels it `(inferred)`. |
| **machines** | When you've coded on more than one machine in the past 7 days. |
| **top editor** | *(verbose)* Most-used editor. Enable with `-v`. |
| **top category** | *(verbose)* coding / debugging / building / etc. Enable with `-v`. |

## Repository layout

| Path | Contents |
|---|---|
| `main.go` | The whole CLI: logos, color schemes, layout engine, Hackatime client, PNG/JPG/SVG export, watch mode. |
| `assets/` | Embedded resources: DejaVu Sans Mono (Bitstream Vera License) used for raster export. |
| `install.sh` | POSIX shell installer for Linux and macOS. Auto-installs prereqs via the system package manager. |
| `install.ps1` | PowerShell installer for Windows (10/11, amd64 and arm64). |
| `Formula/hackfetch.rb` | Homebrew formula (used by the `xerneas3318/tap` tap). |
| `.github/workflows/release.yml` | CI: builds 6 cross-platform binaries on every tag and publishes the GitHub release. |
| `Images/` | Gallery screenshots used in this README. |

## Status

`v1.5.0` is the current release: Linux, macOS, and Windows binaries on every tag, POSIX-compatible installer that auto-installs prereqs across seven package managers, a PowerShell installer for Windows, six built-in Hack Club logos, live `-watch` mode, PNG/JPG/SVG card export, custom color themes, and Hackatime integration with smart language inference.

Related Hack Club tooling and inspirations:

- [Stardance](https://stardance.hackclub.com), the Hack Club hackathon hackfetch was built for.
- [Hackatime](https://hackatime.hackclub.com), Hack Club's WakaTime-compatible coding-time backend.
- [Hack Club](https://hackclub.com), the worldwide community of teen hackers.
- [nFetch](https://github.com/aaronsbytes/nfetch), the dependency-free Go system-fetch that inspired the architecture here.
- [neofetch](https://github.com/dylanaraps/neofetch), the original genre-defining fetch.

## License

[PolyForm Noncommercial 1.0.0](LICENSE). Fork it and tinker for fun, just don't sell it.

---

Maintained by [@xerneas3318](https://github.com/xerneas3318).
