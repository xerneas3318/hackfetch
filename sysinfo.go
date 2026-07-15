package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func getShell() string {
	s := os.Getenv("SHELL")
	if s == "" {
		return missing
	}
	return filepath.Base(s)
}

func getTerm() string {
	return envOr([]string{"TERM_PROGRAM", "TERM"}, missing)
}

func getEditor() string {
	for _, e := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(e); v != "" {
			return filepath.Base(v)
		}
	}
	return missing
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
