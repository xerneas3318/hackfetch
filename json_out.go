package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// machine readable dump of everything hackfetch fetched
// for people who want to pipe into scripts / status bars / wallpaper generators / whatever
// stable enough for shell tooling to grep on top of

type jsonSystem struct {
	User    string `json:"user"`
	Host    string `json:"host"`
	OS      string `json:"os"`
	Shell   string `json:"shell"`
	Term    string `json:"term"`
	Editor  string `json:"editor"`
	Machine string `json:"machine,omitempty"`
}

type jsonProject struct {
	Name    string  `json:"name"`
	Seconds float64 `json:"seconds"`
}

type jsonHackatime struct {
	Slack        string         `json:"slack,omitempty"`
	Host         string         `json:"host,omitempty"`
	Streak       int            `json:"streak"`
	StardanceDays int           `json:"stardance_days_left,omitempty"`
	TodaySeconds float64        `json:"today_seconds"`
	WeekSeconds  float64        `json:"week_seconds"`
	TopProject   *jsonProject   `json:"top_project,omitempty"`
	TopLang      *jsonProject   `json:"top_language,omitempty"`
	WeekChart    []float64      `json:"week_chart,omitempty"`
	Machines     int            `json:"machines,omitempty"`
}

type jsonOut struct {
	Version   string        `json:"version"`
	FetchedAt time.Time     `json:"fetched_at"`
	System    jsonSystem    `json:"system"`
	Hackatime *jsonHackatime `json:"hackatime,omitempty"`
}

// dumps the current fetch as pretty printed json to stdout
// pulls from the same cached responses the normal render uses so its consistent
func printJSON(cfg *config, noNet bool) {
	out := jsonOut{
		Version:   "2",
		FetchedAt: time.Now(),
		System: jsonSystem{
			User:   getUser(),
			Host:   getHost(),
			OS:     getOS(),
			Shell:  getShell(),
			Term:   getTerm(),
			Editor: getEditor(),
		},
	}

	if cfg != nil && !noNet {
		ht := &jsonHackatime{
			Host:   apiHostName(cfg),
			Streak: getStreak(cfg),
		}
		if u := cachedFetchUser(cfg); u != nil {
			ht.Slack = "@" + u.Username
		}
		if days, ok := daysUntilStardanceEnds(); ok {
			ht.StardanceDays = days
		}
		if t := cachedFetchToday(cfg); t != nil {
			ht.TodaySeconds = t.Data.GrandTotal.Seconds
			if best, secs := bestItem(t.Data.Projects); best != "" {
				ht.TopProject = &jsonProject{Name: best, Seconds: secs}
			}
			if best, secs := bestItem(t.Data.Languages); best != "" {
				ht.TopLang = &jsonProject{Name: best, Seconds: secs}
			}
		}
		if w := cachedFetchWeek(cfg); w != nil {
			ht.WeekSeconds = w.Data.TotalSeconds
			if best, secs := bestItem(w.Data.Projects); best != "" && ht.TopProject == nil {
				ht.TopProject = &jsonProject{Name: best, Seconds: secs}
			}
			if best, secs := bestItem(w.Data.Languages); best != "" && ht.TopLang == nil {
				ht.TopLang = &jsonProject{Name: best, Seconds: secs}
			}
			if len(w.Data.Machines) > 1 {
				ht.Machines = len(w.Data.Machines)
			}
		}
		// per-day chart bucketed same way sparkline does
		if u := cachedFetchUser(cfg); u != nil {
			spans := fetchSpans(cfg, u.Username)
			if len(spans) > 0 {
				perDay := make([]float64, 7)
				labels := make([]string, 7)
				now := time.Now()
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
					}
				}
				ht.WeekChart = perDay
			}
		}
		out.Hackatime = ht
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(&out)
}

// most used item name + duration
// skips the unknown/other buckets same as the terminal render
func bestItem(items []item) (string, float64) {
	best := ""
	max := 0.0
	for _, x := range items {
		n := x.Name
		if n == "" || strings.EqualFold(n, "unknown") || strings.EqualFold(n, "other") {
			continue
		}
		if x.TotalSeconds > max {
			max = x.TotalSeconds
			best = n
		}
	}
	return best, max
}

// pulls just the hostname out of cfg.APIURL
// used in both json and normal fetch (the normal render inlines this)
func apiHostName(cfg *config) string {
	if cfg == nil {
		return ""
	}
	h := cfg.APIURL
	for _, p := range []string{"https://", "http://"} {
		if len(h) > len(p) && h[:len(p)] == p {
			h = h[len(p):]
			break
		}
	}
	for i := 0; i < len(h); i++ {
		if h[i] == '/' {
			return h[:i]
		}
	}
	return h
}
