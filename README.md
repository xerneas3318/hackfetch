# hackfetch

A Hack Club themed system fetch with live [Hackatime](https://hackatime.hackclub.com) stats. Shows your machine info next to a customizable logo, plus your today/weekly hours, top project, and top language from Hackatime.

Built for [Stardance](https://stardance.hackclub.com).

```
██   ██  █████   ██████ ██   ██     xerneas@laptop
██   ██ ██   ██ ██      ██  ██      ──────────────
███████ ███████ ██      █████       os           macOS 26.5.1
██   ██ ██   ██ ██      ██  ██      shell        zsh
██   ██ ██   ██  ██████ ██   ██     editor       nvim
                                    hackatime    hackatime.hackclub.com
 ██████ ██      ██    ██ ██████     today        3h 23m
██      ██      ██    ██ ██   ██    7-day total  6h 46m
██      ██      ██    ██ ██████     top project  makemore (3h 27m)
██      ██      ██    ██ ██   ██    top lang     Python (2h 14m)
 ██████ ███████  ██████  ██████
```

## Install

### Homebrew

```sh
brew tap xerneas3318/tap
brew install --HEAD hackfetch
```

### Go

```sh
go install github.com/xerneas3318/hackfetch@latest
```

### From source

```sh
git clone https://github.com/xerneas3318/hackfetch
cd hackfetch
go build -o hackfetch .
```

## Setup

hackfetch reads `~/.wakatime.cfg`. If you don't have one yet, run the official Hack Club setup:

```sh
curl -fsSL https://raw.githubusercontent.com/hackclub/hackatime-setup/main/install.sh | bash
```

Or run `hackfetch -setup` and follow the prompt.

## Usage

```sh
hackfetch                              # defaults
hackfetch stardance rainbow            # positional shorthand
hackfetch logo flag color pride        # keyword form
hackfetch -logo orpheus -color ocean   # flag form
hackfetch -list                        # show all logos + colors
hackfetch -h                           # help
hackfetch -setup                       # (re-)configure hackatime
hackfetch -no-net                      # skip api calls
```

**Logos:** `hackclub` `stardance` `flag` `orpheus` `rocket`

**Colors:** `hackclub` `orange` `mono` `mute` `matrix` `rainbow` `pride` `sunset` `ocean` `forest` `stardance` `trans`

### Set defaults

```sh
export HACKFETCH_LOGO=stardance
export HACKFETCH_COLOR=rainbow
```

## Links

- [Stardance](https://stardance.hackclub.com) — the Hack Club hackathon this was built for
- [Hackatime](https://hackatime.hackclub.com) — Hack Club's WakaTime-compatible backend
- [Hack Club](https://hackclub.com) — the worldwide community of teen hackers
- [neofetch](https://github.com/dylanaraps/neofetch) — the original system fetch this is inspired by

## License

MIT
