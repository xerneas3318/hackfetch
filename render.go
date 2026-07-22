package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// one row of the info column next to the logo
type field struct {
	label, value string
}

// pads with spaces so printable width is at least w runes
func padRunes(s string, w int) string {
	n := utf8.RuneCountInString(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

// colored logo on the left info on the right
func render(logoLines []string, sch scheme, fields []field) {
	const lw = 33
	rows := len(logoLines)
	if len(fields) > rows {
		rows = len(fields)
	}
	fmt.Println()
	for i := 0; i < rows; i++ {
		left := strings.Repeat(" ", lw)
		if i < len(logoLines) {
			left = padRunes(logoLines[i], lw)
		}
		fmt.Print(colorize(left, sch, i))
		if i < len(fields) {
			f := fields[i]
			if f.label == "" {
				fmt.Printf("   %s", f.value)
			} else {
				fmt.Printf("   %s%-12s%s %s%s%s", labelColor(sch, i), f.label, reset, txt, f.value, reset)
			}
		}
		fmt.Println()
	}
	fmt.Println()
}


// one line for tmux / lualine / whatever
// no colors on purpose let the host theme handle styling
// no output at all if theres nothing meaningful so an unconfigured user doesnt see garbage
// includeSpark tacks on the 7 day sparkline for -status --sparkline
func printStatus(cfg *config, includeSpark bool) {
	parts := []string{}
	if t := cachedFetchToday(cfg); t != nil {
		if t.Data.GrandTotal.Seconds > 0 {
			parts = append(parts, fmtDur(t.Data.GrandTotal.Seconds)+" today")
		}
		if tp := topItem(t.Data.Projects); tp != "" {
			parts = append(parts, tp)
		}
	}
	if s := getStreak(cfg); s > 0 {
		parts = append(parts, fmt.Sprintf("streak %d", s))
	}
	if includeSpark {
		if bars, ok := sparklineFieldValue(cfg); ok {
			parts = append(parts, bars)
		}
	}
	if len(parts) > 0 {
		fmt.Println(strings.Join(parts, " · "))
	}
}

// map keys in stable alphabetical order
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// builds the info column shared by render + card exporters
// so the terminal fetch and exported image show the same data
func buildFields(cfg *config, noNet, verbose bool) []field {
	user := getUser()
	host := getHost()
	headline := fmt.Sprintf("%s@%s", user, host)

	fields := []field{
		{"", bold + white + headline + reset},
		{"", dim + strings.Repeat("─", utf8.RuneCountInString(headline)) + reset},
		{"os", getOS()},
		{"shell", getShell()},
		{"term", getTerm()},
		{"editor", getEditor()},
	}

	if cfg != nil && !noNet {
		hostName := strings.TrimPrefix(strings.TrimPrefix(cfg.APIURL, "https://"), "http://")
		if i := strings.Index(hostName, "/"); i >= 0 {
			hostName = hostName[:i]
		}
		fields = append(fields, field{"hackatime", hostName})
		gotAny := false
		if u := cachedFetchUser(cfg); u != nil {
			gotAny = true
			if verbose {
				fields = append(fields, field{"slack", "@" + u.Username})
			}
		}
		if s := getStreak(cfg); s > 0 {
			gotAny = true
			fields = append(fields, field{"streak", formatStreak(s)})
		}
		if dust, ok := getStardust(); ok {
			gotAny = true
			fields = append(fields, field{"stardust", fmt.Sprintf("%d ✦", dust)})
		}
		if days, ok := daysUntilStardanceEnds(); ok {
			gotAny = true
			fields = append(fields, field{"stardance", fmt.Sprintf("%d days left", days)})
		}
		if t := cachedFetchToday(cfg); t != nil {
			gotAny = true
			fields = append(fields, field{"today", fmtDur(t.Data.GrandTotal.Seconds)})
			if tp := topItem(t.Data.Projects); tp != "" {
				fields = append(fields, field{"project", tp})
			}
			if tl := topItem(t.Data.Languages); tl != "" {
				fields = append(fields, field{"language", tl})
			} else if len(t.Data.Languages) > 0 {
				if il, ifc := getInferredLang(cfg); il != "" {
					fields = append(fields, field{"language", fmt.Sprintf("%s%s (~%d files, inferred)%s", il, dim, ifc, reset)})
				} else if n := countUniqueFilesToday(cfg); n > 0 {
					fields = append(fields, field{"files today", fmt.Sprintf("%d", n)})
				}
			}
			if verbose {
				if te := topItem(t.Data.Editors); te != "" {
					fields = append(fields, field{"editor used", te})
				}
				if tc := topItem(t.Data.Categories); tc != "" {
					fields = append(fields, field{"category", tc})
				}
			}
		}
		if w := cachedFetchWeek(cfg); w != nil {
			gotAny = true
			fields = append(fields, field{"7-day total", fmtDur(w.Data.TotalSeconds)})
			if verbose {
				if bars, ok := sparklineFieldValue(cfg); ok {
					fields = append(fields, field{"7-day chart", bars})
				}
			}
			if tp := topItem(w.Data.Projects); tp != "" {
				fields = append(fields, field{"top project", tp})
			}
			if tl := topItem(w.Data.Languages); tl != "" {
				fields = append(fields, field{"top lang", tl})
			} else if len(w.Data.Languages) > 0 {
				if il, ifc := getInferredLang(cfg); il != "" {
					fields = append(fields, field{"top lang", fmt.Sprintf("%s%s (~%d files, inferred)%s", il, dim, ifc, reset)})
				}
			}
			if verbose {
				if te := topItem(w.Data.Editors); te != "" {
					fields = append(fields, field{"top editor", te})
				}
				if tc := topItem(w.Data.Categories); tc != "" {
					fields = append(fields, field{"top category", tc})
				}
			}
			if verbose && len(w.Data.Machines) > 1 {
				fields = append(fields, field{"machines", fmt.Sprintf("%d", len(w.Data.Machines))})
			}
		}
		if !gotAny && lastAPIErr != nil {
			msg := lastAPIErr.Error()
			if strings.Contains(msg, "401") {
				msg = "auth failed. rotate key at hackatime.hackclub.com/my/settings/privacy or run `hackfetch -setup`"
			}
			fields = append(fields, field{"⚠ api", dim + msg + reset})
		}
	} else if noNet {
		fields = append(fields, field{"hackatime", dim + "offline (-no-net)" + reset})
	}

	return fields
}

// wipes memoized api data between watch refreshes
// also flips the dead endpoint short circuits back on in case the server comes back mid session
func resetCaches() {
	hbsCached = false
	cachedHbs = nil
	streakCached = false
	cachedStreak = 0
	userCached = false
	cachedUser = nil
	todayCached = false
	cachedTodayVal = nil
	weekCached = false
	cachedWeekVal = nil
	spansCached = false
	cachedSpans = nil
	cachedSpansUID = ""
	heartbeatsEndpointDead = false
	lastAPIErr = nil
}

// redraws on an interval until ctrl-c
func runWatch(logoLines []string, sch scheme, cfg *config, noNet, verbose bool, interval time.Duration) {
	for {
		resetCaches()
		if cfg != nil && !noNet {
			prefetchAll(cfg)
		}
		fmt.Print("\x1b[2J\x1b[H")
		fields := buildFields(cfg, noNet, verbose)
		render(logoLines, sch, fields)
		fmt.Printf("  %sat %s · refreshes every %ds · ctrl-c to quit%s\n\n",
			dim, time.Now().Format("15:04:05"), int(interval.Seconds()), reset)
		time.Sleep(interval)
	}
}

// only animate the spinner when stderr is a real terminal
// otherwise it spams control codes into whatever pipe or file the user is redirecting to
func stderrIsTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// braille spinner on stderr while hackatime is fetching
// stop() clears the line and blocks until the goroutine actually released it
// so nothing prints on top of a stale frame
// no op if stderr isnt a tty
func startSpinner(label string) func() {
	if !stderrIsTTY() {
		return func() {}
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		defer close(done)
		// paint frame 0 immediately so a fast fetch still flashes something
		fmt.Fprintf(os.Stderr, "\r\x1b[2m%s %s\x1b[0m", frames[0], label)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 1
		for {
			select {
			case <-stop:
				fmt.Fprint(os.Stderr, "\r\x1b[K")
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r\x1b[2m%s %s\x1b[0m", frames[i%len(frames)], label)
				i++
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}
