package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// normalizes whatever the user typed into a bare hackatime username
// accepts @name, name, or a full profile url
// hackatime.hackclub.com/@sampoder -> sampoder
func normalizeUsername(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "hackatime.hackclub.com/")
	s = strings.TrimPrefix(s, "@")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return s
}

// one users totals so we can render two side by side
type vsStats struct {
	username string
	todaySec float64
	weekSec  float64
	totalSec float64
	streak   int
	hasData  bool
}

// buckets a spans slice into today / rolling 7 days / all time totals
// day boundaries are local because hackatime uses server time and thats close enough for a compare
func statsFromSpans(spans []span) (today, week, total float64) {
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	weekStart := now.AddDate(0, 0, -7).Unix()
	for _, s := range spans {
		if s.Duration <= 0 {
			continue
		}
		total += s.Duration
		if int64(s.StartTime) >= weekStart {
			week += s.Duration
		}
		if int64(s.StartTime) >= dayStart {
			today += s.Duration
		}
	}
	return
}

// same streak algorithm as computeStreak but from any spans slice
// consecutive active days ending today or yesterday
func streakFromSpans(spans []span) int {
	if len(spans) == 0 {
		return 0
	}
	active := activeDaysFromSpans(spans)
	cur := time.Now()
	if !active[cur.Format("2006-01-02")] {
		cur = cur.AddDate(0, 0, -1)
		if !active[cur.Format("2006-01-02")] {
			return 0
		}
	}
	streak := 0
	for active[cur.Format("2006-01-02")] {
		streak++
		cur = cur.AddDate(0, 0, -1)
	}
	return streak
}

// direct spans call for a given username
// bypasses fetchSpans cache on purpose so pulling their history doesnt stomp our own
func fetchVsSpans(cfg *config, username string) []span {
	if username == "" {
		return nil
	}
	var r struct {
		Spans []span `json:"spans"`
	}
	url := cfg.nativeBase() + "/users/" + username + "/heartbeats/spans"
	if err := apiGetURL(cfg, url, &r); err != nil {
		dbg("fetchVsSpans %s error %v", username, err)
		return nil
	}
	return r.Spans
}

func computeVsStats(cfg *config, username string) vsStats {
	spans := fetchVsSpans(cfg, username)
	today, week, total := statsFromSpans(spans)
	return vsStats{
		username: username,
		todaySec: today,
		weekSec:  week,
		totalSec: total,
		streak:   streakFromSpans(spans),
		hasData:  len(spans) > 0,
	}
}

// prints a side by side table of your stats vs theirs
// each row has a winner arrow so its instantly readable
// both fetches run in parallel because they hit the same endpoint on the same host
func printVs(cfg *config, otherRaw string) {
	other := normalizeUsername(otherRaw)
	if other == "" {
		fmt.Fprintln(os.Stderr, "vs: no username provided")
		return
	}

	stop := startSpinner("fetching head to head...")

	var meStats, themStats vsStats
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		u := cachedFetchUser(cfg)
		if u == nil || u.Username == "" {
			return
		}
		meStats = computeVsStats(cfg, u.Username)
	}()
	go func() {
		defer wg.Done()
		themStats = computeVsStats(cfg, other)
	}()
	wg.Wait()

	stop()

	if !themStats.hasData {
		fmt.Fprintf(os.Stderr, "no public stats for @%s.\n", other)
		fmt.Fprintln(os.Stderr, "  they might have coding activity set to private, or the username is different from what shows on the leaderboard.")
		fmt.Fprintln(os.Stderr, "  try their exact hackatime username (visible at hackatime.hackclub.com/@<name>).")
		return
	}
	if !meStats.hasData {
		fmt.Fprintln(os.Stderr, "cant find your own hackatime data. try running hackfetch normally first.")
		return
	}

	fmt.Println()
	fmt.Printf("  %s✦ head to head%s  %syou%s  %svs%s  %s@%s%s\n",
		orange, reset,
		white, reset,
		dim, reset,
		white, themStats.username, reset)
	fmt.Println()

	printVsDurRow("today", meStats.todaySec, themStats.todaySec)
	printVsDurRow("week", meStats.weekSec, themStats.weekSec)
	printVsDurRow("all time", meStats.totalSec, themStats.totalSec)
	printVsIntRow("streak", meStats.streak, themStats.streak, "days")
	fmt.Println()

	// a fun little tiebreaker line
	mePoints := 0
	themPoints := 0
	if meStats.todaySec > themStats.todaySec {
		mePoints++
	} else if themStats.todaySec > meStats.todaySec {
		themPoints++
	}
	if meStats.weekSec > themStats.weekSec {
		mePoints++
	} else if themStats.weekSec > meStats.weekSec {
		themPoints++
	}
	if meStats.totalSec > themStats.totalSec {
		mePoints++
	} else if themStats.totalSec > meStats.totalSec {
		themPoints++
	}
	if meStats.streak > themStats.streak {
		mePoints++
	} else if themStats.streak > meStats.streak {
		themPoints++
	}

	var verdict string
	switch {
	case mePoints > themPoints:
		verdict = fmt.Sprintf("  %s🏆 you win %d-%d%s", green, mePoints, themPoints, reset)
	case themPoints > mePoints:
		verdict = fmt.Sprintf("  %s@%s wins %d-%d%s", orange, themStats.username, themPoints, mePoints, reset)
	default:
		verdict = fmt.Sprintf("  %sits a tie %d-%d%s", dim, mePoints, themPoints, reset)
	}
	fmt.Println(verdict)
	fmt.Println()
}

func printVsDurRow(label string, mine, theirs float64) {
	m := fmtDur(mine)
	t := fmtDur(theirs)
	arrow := ""
	switch {
	case mine > theirs:
		arrow = "  " + green + "← you by " + fmtDur(mine-theirs) + reset
	case theirs > mine:
		arrow = "  " + orange + "← them by " + fmtDur(theirs-mine) + reset
	}
	fmt.Printf("  %s%-10s%s  %-14s%-14s%s\n", dim, label, reset, m, t, arrow)
}

func printVsIntRow(label string, mine, theirs int, unit string) {
	m := fmt.Sprintf("%d %s", mine, unit)
	t := fmt.Sprintf("%d %s", theirs, unit)
	arrow := ""
	switch {
	case mine > theirs:
		arrow = "  " + green + fmt.Sprintf("← you by %d", mine-theirs) + reset
	case theirs > mine:
		arrow = "  " + orange + fmt.Sprintf("← them by %d", theirs-mine) + reset
	}
	fmt.Printf("  %s%-10s%s  %-14s%-14s%s\n", dim, label, reset, m, t, arrow)
}
