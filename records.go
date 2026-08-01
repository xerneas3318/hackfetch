package main

import (
	"fmt"
	"os"
	"sort"
	"time"
)

// personal records pulled entirely from the spans response we already fetch
// zero new api calls just math on the same data buildFields consumes
// biggest day, longest streak ever, longest single session, days coded, first day, lifetime hours

type records struct {
	biggestDaySeconds float64
	biggestDayDate    string
	longestStreak     int
	longestStreakEnd  string
	longestSession    float64
	longestSessionAt  string
	daysCoded         int
	firstDay          string
	lifetimeSeconds   float64
}

// crunches the numbers on a spans slice
// all timestamps are unix seconds and interpreted in local time so "biggest day" matches what a human remembers
func computeRecords(spans []span) records {
	if len(spans) == 0 {
		return records{}
	}

	perDay := map[string]float64{}
	var lifetime, longestSession float64
	var longestSessionAt string
	var earliest float64

	for _, s := range spans {
		if s.Duration <= 0 {
			continue
		}
		lifetime += s.Duration
		day := time.Unix(int64(s.StartTime), 0).Format("2006-01-02")
		perDay[day] += s.Duration

		if s.Duration > longestSession {
			longestSession = s.Duration
			longestSessionAt = day
		}
		if earliest == 0 || s.StartTime < earliest {
			earliest = s.StartTime
		}
	}

	var biggestDay float64
	var biggestDate string
	for d, secs := range perDay {
		if secs > biggestDay {
			biggestDay, biggestDate = secs, d
		}
	}

	longest, endedOn := longestStreakInDays(perDay)

	firstDay := ""
	if earliest > 0 {
		firstDay = time.Unix(int64(earliest), 0).Format("2006-01-02")
	}

	return records{
		biggestDaySeconds: biggestDay,
		biggestDayDate:    biggestDate,
		longestStreak:     longest,
		longestStreakEnd:  endedOn,
		longestSession:    longestSession,
		longestSessionAt:  longestSessionAt,
		daysCoded:         len(perDay),
		firstDay:          firstDay,
		lifetimeSeconds:   lifetime,
	}
}

// walks all active days in order and finds the longest consecutive run
// returns the length and the last day of that run so users can go "oh yeah that week"
func longestStreakInDays(perDay map[string]float64) (int, string) {
	if len(perDay) == 0 {
		return 0, ""
	}
	days := make([]string, 0, len(perDay))
	for d := range perDay {
		days = append(days, d)
	}
	sort.Strings(days)

	best, current := 1, 1
	bestEnd := days[0]
	prev, _ := time.Parse("2006-01-02", days[0])
	for i := 1; i < len(days); i++ {
		cur, _ := time.Parse("2006-01-02", days[i])
		if cur.Sub(prev).Hours() <= 24.5 && cur.Sub(prev).Hours() >= 23.5 {
			current++
			if current > best {
				best, bestEnd = current, days[i]
			}
		} else {
			current = 1
		}
		prev = cur
	}
	return best, bestEnd
}

// human "270 days ago" style delta for the first day line
// falls back to just the date if parsing fails for any reason
func daysAgo(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	d := int(time.Since(t).Hours() / 24)
	if d <= 0 {
		return "today"
	}
	if d == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", d)
}

func printRecords(cfg *config) {
	u := cachedFetchUser(cfg)
	if u == nil || u.Username == "" {
		fmt.Fprintln(os.Stderr, "cant find your hackatime user. try hackfetch normally first.")
		return
	}

	stop := startSpinner("digging through your history...")
	spans := fetchSpans(cfg, u.Username)
	stop()

	if len(spans) == 0 {
		fmt.Fprintln(os.Stderr, "no spans yet. code a bit and try again.")
		return
	}

	r := computeRecords(spans)

	fmt.Println()
	fmt.Printf("  %s✦ personal records%s  %s@%s%s\n", orange, reset, white, u.Username, reset)
	fmt.Println()

	printRecordRow("biggest day", fmtDur(r.biggestDaySeconds), "on "+r.biggestDayDate)
	printRecordRow("longest streak", formatStreak(r.longestStreak), "ended "+r.longestStreakEnd)
	printRecordRow("longest session", fmtDur(r.longestSession), "on "+r.longestSessionAt)
	printRecordRow("days coded", fmt.Sprintf("%d", r.daysCoded), "")
	printRecordRow("first day", r.firstDay, daysAgo(r.firstDay))
	printRecordRow("lifetime", fmtDur(r.lifetimeSeconds), "")
	fmt.Println()
}

// aligned record row: label on the left value in the middle a dim aside on the right
func printRecordRow(label, value, aside string) {
	if aside == "" {
		fmt.Printf("  %s%-16s%s  %s%s%s\n", dim, label, reset, txt, value, reset)
		return
	}
	fmt.Printf("  %s%-16s%s  %s%-14s%s  %s%s%s\n", dim, label, reset, txt, value, reset, dim, aside, reset)
}
