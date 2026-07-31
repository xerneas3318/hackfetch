package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// placeholder when an env var or system call returns nothing
// short so the info column stays tidy
const missing = "-"

// seconds -> compact "1h 23m" style
func fmtDur(seconds float64) string {
	if seconds < 1 {
		return "0m"
	}
	h := int(seconds / 3600)
	m := int((seconds - float64(h)*3600) / 60)
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// first non empty env var in keys or fallback
func envOr(keys []string, fallback string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return fallback
}

// current shell not the login shell
// $SHELL only tells us the login shell so someone who logs in as bash but runs fish
// would show up as bash. fingerprint env vars are the cheapest tell then fall back to
// asking the OS who our parent process is
func getShell() string {
	if os.Getenv("FISH_VERSION") != "" {
		return "fish"
	}
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	}
	if os.Getenv("BASH_VERSION") != "" {
		return "bash"
	}
	if name := parentShellName(); name != "" {
		return name
	}
	if s := os.Getenv("SHELL"); s != "" {
		return filepath.Base(s)
	}
	return missing
}

// asks the OS who our parent process is
// login shells sometimes prefix themselves with a dash (-bash) strip it
// only returns known shell names so we dont accidentally report "tmux" or "sshd" or "kitty"
func parentShellName() string {
	ppid := os.Getppid()
	if ppid <= 1 {
		return ""
	}
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(ppid)).Output()
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	name = strings.TrimPrefix(name, "-")
	name = filepath.Base(name)
	name = strings.TrimSuffix(strings.ToLower(name), ".exe")
	switch name {
	case "sh", "bash", "zsh", "fish", "ksh", "dash", "tcsh", "csh", "elvish", "nushell", "nu", "xonsh", "pwsh", "powershell":
		return name
	}
	return ""
}

func getTerm() string {
	return envOr([]string{"TERM_PROGRAM", "TERM"}, missing)
}

// $VISUAL then $EDITOR
// stripping trailing args like "code --wait" and mapping the ugly launcher names
// (zeditor is really Zed, code is really VS Code, etc.) so the fetch reads like english
func getEditor() string {
	for _, e := range []string{"VISUAL", "EDITOR"} {
		if v := strings.TrimSpace(os.Getenv(e)); v != "" {
			return prettifyEditor(v)
		}
	}
	return missing
}

// splits on whitespace to drop args, lowercases, drops .exe, then maps known launchers
// unknown names pass through unchanged so custom editors still show up
func prettifyEditor(v string) string {
	if i := strings.IndexAny(v, " \t"); i >= 0 {
		v = v[:i]
	}
	base := strings.ToLower(filepath.Base(v))
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "zeditor", "zed":
		return "Zed"
	case "code":
		return "VS Code"
	case "code-insiders":
		return "VS Code Insiders"
	case "codium", "vscodium":
		return "VSCodium"
	case "cursor":
		return "Cursor"
	case "windsurf":
		return "Windsurf"
	case "subl", "sublime_text":
		return "Sublime Text"
	case "atom":
		return "Atom"
	case "hx", "helix":
		return "Helix"
	case "nvim":
		return "Neovim"
	case "vim":
		return "Vim"
	case "vi":
		return "vi"
	case "emacs":
		return "Emacs"
	case "kak":
		return "Kakoune"
	case "micro":
		return "micro"
	case "nano":
		return "nano"
	case "pico":
		return "pico"
	case "ed":
		return "ed"
	}
	return base
}

// human os label
// on macos shell out to sw_vers for the version because runtime.GOOS only gives us "darwin"
func getOS() string {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			return "macOS " + strings.TrimSpace(string(out))
		}
		return "macOS"
	}
	return runtime.GOOS
}

func getHost() string {
	h, err := os.Hostname()
	if err != nil {
		return missing
	}
	return strings.TrimSuffix(h, ".local")
}

func getUser() string {
	return envOr([]string{"USER", "LOGNAME"}, missing)
}
