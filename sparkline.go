package main

import (
	"fmt"
	"strings"
	"time"
)

// 8 levels of bars
// index 0 is a space so empty days dont show a stub
var sparkBars = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}


// 7 char sparkline of the last week
// oldest on the left newest on the right
// piggybacks on the same spans call computeStreak already makes so its basically free
func buildSparkline(cfg *config) (bars string, total float64) {
	u := cachedFetchUser(cfg)
	if u == nil || u.Username == "" {
		return "", 0
	}
	spans := fetchSpans(cfg, u.Username)
	if len(spans) == 0 {
		return "", 0
	}

	// bucket into the last 7 local days
	now := time.Now()
	perDay := make([]float64, 7)
	labels := make([]string, 7)
	for i := 0; i < 7; i++ {
		labels[i] = now.AddDate(0, 0, -(6 - i)).Format("2006-01-02")
	}
	idx := map[string]int{}
	for i, l := range labels {
		idx[l] = i
	}
	for _, s := range spans {
		if s.Duration <= 0 {
			continue
		}
		day := time.Unix(int64(s.StartTime), 0).Format("2006-01-02")
		if i, ok := idx[day]; ok {
			perDay[i] += s.Duration
			total += s.Duration
		}
	}

	// scale against the peak day
	// TODO log scale would be prettier on lopsided weeks but eh good enough
	var max float64
	for _, v := range perDay {
		if v > max {
			max = v
		}
	}
	var b strings.Builder
	for _, v := range perDay {
		switch {
		case v <= 0:
			b.WriteRune(sparkBars[0])
		case max <= 0:
			b.WriteRune(sparkBars[1])
		default:
			// any active day gets at least 1 bar
			step := int((v/max)*7 + 0.5)
			if step < 1 {
				step = 1
			}
			if step > 8 {
				step = 8
			}
			b.WriteRune(sparkBars[step])
		}
	}
	return b.String(), total
}

// bars + weekly total for the fetch field / -status suffix
// ok = false when theres nothing to draw (no net or fresh account)
func sparklineFieldValue(cfg *config) (string, bool) {
	bars, total := buildSparkline(cfg)
	if bars == "" {
		return "", false
	}
	return fmt.Sprintf("%s %s", bars, fmtDur(total)), true
}
