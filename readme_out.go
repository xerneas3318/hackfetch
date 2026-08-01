package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// spits out a markdown snippet ready to paste into a github profile README
// pulls from the same cached responses buildFields uses so this is basically free after a normal fetch
// no ansi no colors just plain markdown a github renderer will happily eat
func printReadmeMarkdown(cfg *config, noNet bool) {
	var b strings.Builder

	b.WriteString("### hackatime stats\n\n")

	if cfg == nil || noNet {
		b.WriteString("_hackfetch was run without a hackatime config so theres nothing to show yet._\n\n")
		b.WriteString(footerLine())
		fmt.Print(b.String())
		return
	}

	b.WriteString("| stat | value |\n")
	b.WriteString("|------|-------|\n")

	if t := cachedFetchToday(cfg); t != nil {
		b.WriteString(fmt.Sprintf("| today | %s |\n", fmtDur(t.Data.GrandTotal.Seconds)))
	}
	if w := cachedFetchWeek(cfg); w != nil {
		b.WriteString(fmt.Sprintf("| this week | %s |\n", fmtDur(w.Data.TotalSeconds)))
	}
	if s := getStreak(cfg); s > 0 {
		flame := ""
		if s >= 3 {
			flame = " 🔥"
		}
		b.WriteString(fmt.Sprintf("| streak | %s%s |\n", formatStreak(s), flame))
	}
	if t := cachedFetchToday(cfg); t != nil {
		if best, secs := bestItem(t.Data.Projects); best != "" {
			b.WriteString(fmt.Sprintf("| top project today | %s (%s) |\n", mdEscape(best), fmtDur(secs)))
		}
		if best, secs := bestItem(t.Data.Languages); best != "" {
			b.WriteString(fmt.Sprintf("| top language today | %s (%s) |\n", mdEscape(best), fmtDur(secs)))
		}
	}
	if w := cachedFetchWeek(cfg); w != nil {
		if best, secs := bestItem(w.Data.Projects); best != "" {
			b.WriteString(fmt.Sprintf("| top project 7d | %s (%s) |\n", mdEscape(best), fmtDur(secs)))
		}
		if best, secs := bestItem(w.Data.Languages); best != "" {
			b.WriteString(fmt.Sprintf("| top language 7d | %s (%s) |\n", mdEscape(best), fmtDur(secs)))
		}
	}

	// deep link to the users hackatime profile if we know who they are
	if u := cachedFetchUser(cfg); u != nil && u.Username != "" {
		b.WriteString(fmt.Sprintf("\n[full profile on hackatime](https://hackatime.hackclub.com/@%s)\n\n", u.Username))
	} else {
		b.WriteString("\n")
	}

	b.WriteString(footerLine())
	fmt.Print(b.String())

	// helpful reminder to stderr on how to keep it fresh
	// stdout stays pure markdown so `hackfetch -readme > stats.md` produces a clean file
	if stderrIsTTY() {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  "+dim+"tip: pipe this into your profile readme:"+reset)
		fmt.Fprintln(os.Stderr, "  "+dim+"  hackfetch -readme > stats.md"+reset)
		fmt.Fprintln(os.Stderr, "  "+dim+"or wire it into a github action so it refreshes daily"+reset)
	}
}

// prevents a project or language named `some|thing` from breaking the table
func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// stamp so people know the card is fresh (or stale)
func footerLine() string {
	stamp := time.Now().Format("2006-01-02")
	return fmt.Sprintf("_powered by [hackfetch](https://github.com/xerneas3318/hackfetch) v%s · updated %s_\n", version, stamp)
}
