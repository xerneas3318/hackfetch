package main

import (
	"fmt"
	"os"
	"strings"
)

// leaderboard entry from /api/v1/leaderboard/daily and /weekly
// public endpoint no auth needed
type leaderboardEntry struct {
	Rank int `json:"rank"`
	User struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
	TotalSeconds float64 `json:"total_seconds"`
}

type leaderboardResp struct {
	Period    string             `json:"period"`
	DateRange string             `json:"date_range"`
	Entries   []leaderboardEntry `json:"entries"`
}

// hits /api/v1/leaderboard/{period} where period is daily or weekly
// public so no auth required but hackfetch still sends the bearer token because httpClient always does
func fetchLeaderboard(cfg *config, period string) *leaderboardResp {
	if period != "daily" && period != "weekly" {
		period = "daily"
	}
	var r leaderboardResp
	url := cfg.nativeBase() + "/leaderboard/" + period
	if err := apiGetURL(cfg, url, &r); err != nil {
		dbg("fetchLeaderboard error %v", err)
		return nil
	}
	return &r
}

// pretty prints the top N with rank + username + total hours
// each row uses OSC 8 to make the username clickable to hackatime.hackclub.com/@name
// falls back cleanly on terminals that dont support OSC 8 (the link just doesnt render as a link)
// highlights the current user if we can figure out who they are
func printLeaderboard(cfg *config, period string, limit int) {
	lb := fetchLeaderboard(cfg, period)
	if lb == nil || len(lb.Entries) == 0 {
		fmt.Fprintln(os.Stderr, "no leaderboard data")
		return
	}
	if limit <= 0 || limit > len(lb.Entries) {
		limit = len(lb.Entries)
	}

	// figure out who "me" is so we can highlight the row
	me := ""
	if u := cachedFetchUser(cfg); u != nil {
		me = u.Username
	}

	title := "daily leaderboard"
	if lb.Period == "last_7_days" {
		title = "weekly leaderboard"
	}

	fmt.Println()
	fmt.Printf("  %s✦ %s%s  %s%s%s\n", orange, title, reset, dim, lb.DateRange, reset)
	fmt.Println()

	for i := 0; i < limit; i++ {
		e := lb.Entries[i]
		medal := "  "
		switch e.Rank {
		case 1:
			medal = "🥇"
		case 2:
			medal = "🥈"
		case 3:
			medal = "🥉"
		}
		nameCol := white
		suffix := ""
		if me != "" && strings.EqualFold(me, e.User.Username) {
			nameCol = orange
			suffix = " " + dim + "(you)" + reset
		}
		link := osc8Link("https://hackatime.hackclub.com/@"+e.User.Username, e.User.Username)
		fmt.Printf("  %s %s%2d.%s %s%-24s%s %s%s%s\n",
			medal, dim, e.Rank, reset,
			nameCol, link+suffix, reset,
			txt, fmtDur(e.TotalSeconds), reset)
	}
	fmt.Println()
}

// wraps text in an OSC 8 hyperlink escape
// modern terminals (iterm2 kitty wezterm alacritty gnome-terminal) render this as a clickable link
// terminals without OSC 8 just show the text
func osc8Link(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
