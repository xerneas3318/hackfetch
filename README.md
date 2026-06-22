# hackfetch

A Hack Club themed system fetch with live [Hackatime](https://hackatime.hackclub.com) stats. Shows your system info next to a customizable Hack Club logo, plus your today/weekly hours, top project, top language, streak, and more - all from your terminal.

Built for [Stardance](https://stardance.hackclub.com) ✦

<p align="center">
  <img src="Images/stardance-ocean.png" alt="hackfetch stardance ocean" width="720">
</p>

---

## Install

### One-liner — Linux + macOS

```sh
curl -fsSL https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.sh | sh
```

Works on **any major distro** — Ubuntu, Debian, Fedora, RHEL, CentOS, Arch, openSUSE, Alpine — and macOS. The installer:

- detects your OS/arch (Linux x86_64/arm64, macOS Intel/Apple Silicon)
- **auto-installs prereqs** (`curl`, `tar`, `xdg-utils`) via your system's package manager — `apt`, `dnf`, `yum`, `pacman`, `zypper`, `apk`, or `brew`
- downloads the matching pre-built binary from GitHub Releases
- installs to `/usr/local/bin` (using `sudo` if needed) or falls back to `~/.local/bin`
- is POSIX `sh`-compatible, so it works on Alpine/BusyBox and minimal containers too

Override the install location:

```sh
HACKFETCH_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.sh | sh
```

### One-liner — Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.ps1 | iex
```

Works on Windows 10 / 11 (x64 and ARM64). The installer:

- detects your CPU arch
- downloads the matching `hackfetch-windows-<arch>.zip` from GitHub Releases
- installs `hackfetch.exe` to `%LOCALAPPDATA%\Programs\hackfetch`
- adds that folder to your **user** `PATH` (open a new terminal to pick it up)

Override the install location:

```powershell
$env:HACKFETCH_INSTALL_DIR = 'C:\tools\hackfetch'
irm https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.ps1 | iex
```

### Homebrew (macOS + Linuxbrew)

```sh
brew tap xerneas3318/tap
brew install hackfetch
```

### Go (any platform with Go installed)

```sh
go install github.com/xerneas3318/hackfetch@latest
```

---

## Setup

hackfetch reads `~/.wakatime.cfg`. If you don't have one yet, run Hack Club's official Hackatime setup:

```sh
curl -fsSL https://raw.githubusercontent.com/hackclub/hackatime-setup/main/install.sh | bash
```

Or run `hackfetch -setup` and follow the prompt - it'll walk you through opening [hackatime.hackclub.com/my/wakatime_setup](https://hackatime.hackclub.com/my/wakatime_setup) and waiting for the config to be written.

---

## Usage

```sh
hackfetch                              # defaults
hackfetch stardance rainbow            # positional shorthand
hackfetch logo flag color pride        # keyword form
hackfetch -logo orpheus -color ocean   # flag form
hackfetch -v                           # verbose: + top editor, top category
hackfetch -watch                       # live mode, refreshes every 30s
hackfetch -export card.svg             # save the fetch as a shareable SVG card
hackfetch -list                        # show all logos and colors
hackfetch -h                           # help
hackfetch -setup                       # (re-)configure hackatime
hackfetch -no-net                      # offline mode
```

> **Tip:** flags go before positional args. `hackfetch -export card.svg stardance pride` works; `hackfetch stardance pride -export card.svg` doesn't.

### Set defaults

Add to your `~/.zshrc` or `~/.bashrc`:

```sh
export HACKFETCH_LOGO=stardance
export HACKFETCH_COLOR=rainbow
export HACKFETCH_VERBOSE=1
```

---

## Gallery

<table>
  <tr>
    <td align="center">
      <img src="Images/hackclub-forest.png" alt="hackclub forest" width="380"><br>
      <code>hackfetch forest</code>
    </td>
    <td align="center">
      <img src="Images/hackclub-trans.png" alt="hackclub trans" width="380"><br>
      <code>hackfetch trans</code>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="Images/rocket-hackclub.png" alt="rocket" width="380"><br>
      <code>hackfetch rocket</code>
    </td>
    <td align="center">
      <img src="Images/flag-sunset.png" alt="flag sunset" width="380"><br>
      <code>hackfetch flag sunset</code>
    </td>
  </tr>
  <tr>
    <td align="center">
      <img src="Images/bot-matrix.png" alt="bot matrix" width="380"><br>
      <code>hackfetch bot matrix</code>
    </td>
    <td align="center">
      <img src="Images/stardance-ocean.png" alt="stardance ocean" width="380"><br>
      <code>hackfetch stardance ocean</code>
    </td>
  </tr>
</table>

---

## Logos

`hackclub` &nbsp; `stardance` &nbsp; `flag` &nbsp; `orpheus` &nbsp; `bot` &nbsp; `rocket`

## Color schemes

**Basics:** `hackclub` &nbsp; `orange` &nbsp; `mono` &nbsp; `mute` &nbsp; `matrix`

**Gradients & rainbows:** `rainbow` &nbsp; `sunset` &nbsp; `ocean` &nbsp; `forest` &nbsp; `stardance`

**Pride flags:** `pride` &nbsp; `trans` &nbsp; `bi` &nbsp; `pan`

**Special:** `auto` (defaults to `pride` in June, `hackclub` otherwise)

Run `hackfetch -list` for the full set.

---

## Live mode

```sh
hackfetch -watch
```

Re-fetches your Hackatime stats every 30 seconds and redraws in place. Today's hours tick up as you code. Ctrl+C to quit.

## Share your fetch

Export the current fetch as a shareable SVG card:

```sh
hackfetch -export card.svg
hackfetch -export card.svg stardance pride
hackfetch -export card.svg -logo orpheus -color rainbow
```

The output is an SVG with rounded corners, monospace font, and matching colors - perfect for tweeting, posting on Slack, or dropping into a devlog. Open in a browser to view, or convert to PNG with `rsvg-convert` or any image editor.

---

## Custom themes

Define your own color schemes in `~/.config/hackfetch/colors.json`:

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

`mode` can be `single`, `per-line`, or `per-char`. `colors` are [ANSI 256 color codes](https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit). Your themes override built-ins of the same name.

Then run:

```sh
hackfetch -color vaporwave
```

---

## What you get from Hackatime

When your `~/.wakatime.cfg` points at a working Hackatime account, hackfetch fetches and shows:

- **today** - hours coded today
- **7-day total** - hours coded over the past week
- **streak** - consecutive days with activity
- **stardust** ✦ - your stardust count (set via `HACKFETCH_STARDUST=N` env var)
- **stardance** - days left until Stardance ends (auto-hides after Sep 30, 2026)
- **slack** - your Hack Club / Hackatime handle
- **top project / project** - most-worked project (today and weekly)
- **top lang / language** - most-used language (with smart fallback: when Hackatime reports `unknown`, hackfetch infers from file extensions in your heartbeat history and labels it `(inferred)`)
- **machines** - when you've coded on more than one machine in the past 7 days
- **top editor / editor used** - *(verbose)* most-used editor (`-v`)
- **top category / category** - *(verbose)* coding / debugging / building / etc. (`-v`)

---

## Links

- [Stardance](https://stardance.hackclub.com) - the Hack Club hackathon this was built for
- [Hackatime](https://hackatime.hackclub.com) - Hack Club's WakaTime-compatible backend
- [Hack Club](https://hackclub.com) - the worldwide community of teen hackers
- [nFetch](https://github.com/aaronsbytes/nfetch) - fast, dependency-free Go system-fetch that inspired the architecture here
- [neofetch](https://github.com/dylanaraps/neofetch) - the original genre-defining fetch (now archived)

---

## License

MIT
