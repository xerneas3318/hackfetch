package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var debug = os.Getenv("HACKFETCH_DEBUG") != ""

func dbg(format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "[hackfetch] "+format+"\n", args...)
	}
}

const (
	reset = "\x1b[0m"
	bold  = "\x1b[1m"
	white = "\x1b[38;5;231m"
	dim   = "\x1b[38;5;243m"
	txt   = "\x1b[38;5;253m"
	orange = "\x1b[38;5;208m"
	green  = "\x1b[38;5;42m"
)

func ansi(code int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", code)
}

// ─── logos ───────────────────────────────────────────────────────────────────

var logos = map[string][]string{
	"hackclub": {
		"██   ██  █████   ██████ ██   ██",
		"██   ██ ██   ██ ██      ██  ██ ",
		"███████ ███████ ██      █████  ",
		"██   ██ ██   ██ ██      ██  ██ ",
		"██   ██ ██   ██  ██████ ██   ██",
		"                               ",
		" ██████ ██      ██    ██ ██████",
		"██      ██      ██    ██ ██   ██",
		"██      ██      ██    ██ ██████",
		"██      ██      ██    ██ ██   ██",
		" ██████ ███████  ██████  ██████",
	},
	"stardance": {
		"        ·  ✦  ·  ★  ·         ",
		"      ✧               ✧       ",
		"   ★                     ★    ",
		"       ✦             ✦        ",
		"  ✧    s t a r d a n c e   ✧  ",
		"       ✦      2026     ✦      ",
		"   ★                     ★    ",
		"       ✧             ✧        ",
		"      ✦   hack  club   ✦      ",
		"        ·  ★  ·  ✦  ·         ",
		"             ✧                ",
	},
	"flag": {
		"    ╭───────────╮             ",
		"    │░░░░░░░░░░░│             ",
		"    │░ H A C K ░│             ",
		"    │░░░░░░░░░░░│             ",
		"    │░ C L U B ░│             ",
		"    │░░░░░░░░░░░│             ",
		"    ╰─────╮╭────╯             ",
		"          ││                  ",
		"          ││                  ",
		"          ││                  ",
		"         ─┴─                  ",
	},
	"bot": {
		"       ╭────────╮             ",
		"       │ ◉  ─ ◉ │             ",
		"       │   ◡    │             ",
		"       ╰───┬────╯             ",
		"        ╭──┴──╮               ",
		"      ●─┤  ♥  ├─●             ",
		"        │HACK │               ",
		"        │CLUB │               ",
		"        ╰┬───┬╯               ",
		"         ┴   ┴                ",
	},
	"orpheus": {
		"       ╭──╮___                ",
		"       (◉ ___)                ",
		"         │                    ",
		"         │                    ",
		"      ╭──┴──╮                 ",
		"    ●─┤HACK ├─●               ",
		"      │CLUB │                 ",
		"      ╰┬───┬╯                 ",
		"       ┴   ┴                  ",
	},
	"rocket": {
		"           /\\                 ",
		"          /  \\                ",
		"         |    |               ",
		"         |HACK|               ",
		"         |CLUB|               ",
		"         |    |               ",
		"        /|    |\\              ",
		"       / |    | \\             ",
		"      /__|____|__\\            ",
		"          /\\/\\                ",
		"         /////\\               ",
	},
}

// ─── color schemes ───────────────────────────────────────────────────────────

type colorMode int

const (
	modeSingle colorMode = iota
	modePerLine
	modePerChar
)

type scheme struct {
	colors []int
	mode   colorMode
}

var schemes = map[string]scheme{
	"hackclub":  {[]int{203}, modeSingle},
	"orange":    {[]int{208}, modeSingle},
	"mono":      {[]int{231}, modeSingle},
	"mute":      {[]int{243}, modeSingle},
	"matrix":    {[]int{40}, modeSingle},
	"rainbow":   {[]int{196, 202, 208, 214, 220, 226, 190, 154, 118, 82, 46, 49, 51, 45, 39, 33, 27, 93, 129, 165, 201, 199, 197}, modePerChar},
	"pride":     {[]int{196, 208, 226, 46, 27, 129}, modePerLine},
	"sunset":    {[]int{52, 88, 124, 160, 196, 202, 208, 214, 220, 226, 228}, modePerLine},
	"ocean":     {[]int{17, 18, 19, 20, 21, 27, 33, 39, 45, 51, 87}, modePerLine},
	"forest":    {[]int{22, 28, 34, 40, 46, 82, 118, 154, 190}, modePerLine},
	"stardance": {[]int{93, 99, 141, 177, 213, 219, 159, 123, 87, 51, 45}, modePerChar},
	"trans":     {[]int{45, 213, 231, 213, 45}, modePerLine},

	// Pride flag pack
	"bi":          {[]int{199, 99, 27}, modePerLine},
	"bisexual":    {[]int{199, 99, 27}, modePerLine},
	"lesbian":     {[]int{124, 166, 208, 231, 218, 199, 161}, modePerLine},
	"pan":         {[]int{199, 226, 33}, modePerLine},
	"pansexual":   {[]int{199, 226, 33}, modePerLine},
	"nonbinary":   {[]int{226, 231, 99, 240}, modePerLine},
	"nb":          {[]int{226, 231, 99, 240}, modePerLine},
	"ace":         {[]int{240, 248, 231, 99}, modePerLine},
	"asexual":     {[]int{240, 248, 231, 99}, modePerLine},
	"aro":         {[]int{28, 41, 231, 248, 240}, modePerLine},
	"aromantic":   {[]int{28, 41, 231, 248, 240}, modePerLine},
	"agender":     {[]int{240, 248, 231, 41, 231, 248, 240}, modePerLine},
	"genderfluid": {[]int{199, 231, 99, 240, 27}, modePerLine},
	"intersex":    {[]int{226, 99}, modePerLine},
	"demi":        {[]int{240, 248, 231, 99}, modePerLine},
	"poly":        {[]int{27, 196, 240}, modePerLine},
	"progress":    {[]int{196, 208, 226, 46, 27, 129, 240, 218, 45, 231}, modePerLine},
}

func colorize(s string, sch scheme, lineIdx int) string {
	if sch.mode == modeSingle {
		return ansi(sch.colors[0]) + s + reset
	}
	if sch.mode == modePerLine {
		return ansi(sch.colors[lineIdx%len(sch.colors)]) + s + reset
	}
	var b strings.Builder
	i := 0
	for _, r := range s {
		b.WriteString(ansi(sch.colors[(i+lineIdx)%len(sch.colors)]))
		b.WriteRune(r)
		i++
	}
	b.WriteString(reset)
	return b.String()
}

// labelColor picks one ANSI code per line so info labels match the scheme.
func labelColor(sch scheme, lineIdx int) string {
	if len(sch.colors) == 0 {
		return orange
	}
	return ansi(sch.colors[lineIdx%len(sch.colors)])
}

// loadCustomSchemes reads ~/.config/hackfetch/colors.json and merges into schemes.
// User entries override built-ins of the same name.
// Format:
//
//	{ "schemes": { "mytheme": { "colors": [196, 202, 208], "mode": "per-line" } } }
func loadCustomSchemes() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	path := filepath.Join(home, ".config", "hackfetch", "colors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg struct {
		Schemes map[string]struct {
			Colors []int  `json:"colors"`
			Mode   string `json:"mode"`
		} `json:"schemes"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		dbg("custom schemes parse error: %v", err)
		return
	}
	for name, s := range cfg.Schemes {
		if len(s.Colors) == 0 {
			continue
		}
		var mode colorMode
		switch strings.ToLower(s.Mode) {
		case "per-line", "perline", "line", "band":
			mode = modePerLine
		case "per-char", "perchar", "char", "rainbow":
			mode = modePerChar
		default:
			mode = modeSingle
		}
		schemes[name] = scheme{colors: s.Colors, mode: mode}
	}
}

// resolveAutoScheme returns the scheme name when -color auto is requested.
// In Pride Month (June), defaults to pride. Otherwise, hackclub.
func resolveAutoScheme() string {
	if time.Now().Month() == time.June {
		return "pride"
	}
	return "hackclub"
}

// ─── stardance ───────────────────────────────────────────────────────────────

const stardanceEnd = "2026-09-30"

// daysUntilStardanceEnds returns days remaining and true if Stardance is still on.
func daysUntilStardanceEnds() (int, bool) {
	end, err := time.Parse("2006-01-02", stardanceEnd)
	if err != nil {
		return 0, false
	}
	d := int(time.Until(end).Hours() / 24)
	if d < 0 {
		return 0, false
	}
	return d, true
}

// getStardust reads HACKFETCH_STARDUST env var as the user's stardust count.
func getStardust() (int, bool) {
	v := strings.TrimSpace(os.Getenv("HACKFETCH_STARDUST"))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ─── config ──────────────────────────────────────────────────────────────────

type config struct {
	APIKey string
	APIURL string
}

// cfgPath returns the first existing wakatime config, or the default write path if none found.
func cfgPath() (string, error) {
	if h := os.Getenv("WAKATIME_HOME"); h != "" {
		p := filepath.Join(h, ".wakatime.cfg")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(home, ".wakatime.cfg"),
		filepath.Join(home, ".wakatime", ".wakatime.cfg"),
		filepath.Join(home, ".config", "wakatime", ".wakatime.cfg"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return candidates[0], nil
}

// findWakatimeCLI returns the path to wakatime-cli if installed, else "".
func findWakatimeCLI() string {
	if p, err := exec.LookPath("wakatime-cli"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	matches, _ := filepath.Glob(filepath.Join(home, ".wakatime", "wakatime-cli*"))
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return m
		}
	}
	for _, p := range []string{"/usr/local/bin/wakatime-cli", "/opt/homebrew/bin/wakatime-cli"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// configFileFound checks if a wakatime config file exists at any known location.
func configFileFound() (path string, ok bool) {
	p, err := cfgPath()
	if err != nil {
		return "", false
	}
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return p, false
}

func loadConfig() (*config, error) {
	path, err := cfgPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg := &config{APIURL: "https://hackatime.hackclub.com/api/hackatime/v1"}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		switch k {
		case "api_key", "api-key":
			cfg.APIKey = v
		case "api_url":
			cfg.APIURL = strings.TrimSuffix(v, "/")
		}
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no api_key in ~/.wakatime.cfg")
	}
	return cfg, nil
}

func saveAPIKey(key string) error {
	path, err := cfgPath()
	if err != nil {
		return err
	}
	existing, _ := os.ReadFile(path)
	if len(existing) == 0 {
		content := "[settings]\n" +
			"api_url = https://hackatime.hackclub.com/api/hackatime/v1\n" +
			"api_key = " + key + "\n" +
			"heartbeat_rate_limit_seconds = 30\n"
		return os.WriteFile(path, []byte(content), 0600)
	}
	lines := strings.Split(string(existing), "\n")
	replaced := false
	hasSettings := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if t == "[settings]" {
			hasSettings = true
		}
		if strings.HasPrefix(t, "api_key") || strings.HasPrefix(t, "api-key") {
			lines[i] = "api_key = " + key
			replaced = true
		}
	}
	if !replaced {
		if !hasSettings {
			lines = append([]string{"[settings]"}, lines...)
		}
		lines = append(lines, "api_key = "+key)
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0600)
}

// ─── api ─────────────────────────────────────────────────────────────────────

func (c *config) nativeBase() string {
	if i := strings.Index(c.APIURL, "/api/hackatime/"); i >= 0 {
		return c.APIURL[:i] + "/api/v1"
	}
	return c.APIURL
}

func apiGetURL(cfg *config, url string, into any) error {
	dbg("GET %s", url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(cfg.APIKey, "waka_"))
	req.Header.Set("Accept", "application/json")
	c := &http.Client{Timeout: 10 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		dbg("transport error: %v", err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dbg("status=%d body=%.300s", resp.StatusCode, string(body))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, into)
}

func apiGet(cfg *config, path string, into any) error {
	return apiGetURL(cfg, cfg.APIURL+path, into)
}

type userInfo struct {
	Username string
}

var lastAPIErr error

func fetchUser(cfg *config) *userInfo {
	var raw map[string]any
	url := cfg.nativeBase() + "/my/heartbeats/most_recent"
	if err := apiGetURL(cfg, url, &raw); err != nil {
		lastAPIErr = err
		dbg("fetchUser error: %v", err)
		return nil
	}
	if u := findStringField(raw, []string{"username", "user_id", "slack_uid"}); u != "" {
		return &userInfo{Username: u}
	}
	dbg("fetchUser: no username field in response")
	return nil
}

func findStringField(v any, keys []string) string {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range keys {
			if s, ok := t[k].(string); ok && s != "" {
				return s
			}
		}
		for _, child := range t {
			if s := findStringField(child, keys); s != "" {
				return s
			}
		}
	case []any:
		for _, child := range t {
			if s := findStringField(child, keys); s != "" {
				return s
			}
		}
	}
	return ""
}

type item struct {
	Name         string  `json:"name"`
	TotalSeconds float64 `json:"total_seconds"`
}

type todayResp struct {
	Data struct {
		GrandTotal struct {
			Seconds float64 `json:"total_seconds"`
		} `json:"grand_total"`
		Projects   []item `json:"projects"`
		Languages  []item `json:"languages"`
		Editors    []item `json:"editors"`
		Categories []item `json:"categories"`
	} `json:"data"`
}

type weekResp struct {
	Data struct {
		TotalSeconds       float64 `json:"total_seconds"`
		HumanReadableTotal string  `json:"human_readable_total"`
		Projects           []item  `json:"projects"`
		Languages          []item  `json:"languages"`
		Editors            []item  `json:"editors"`
		Categories         []item  `json:"categories"`
		Machines           []item  `json:"machines"`
		OperatingSystems   []item  `json:"operating_systems"`
	} `json:"data"`
}

func fetchToday(cfg *config) *todayResp {
	var r todayResp
	if err := apiGet(cfg, "/users/current/statusbar/today", &r); err != nil {
		dbg("fetchToday error: %v", err)
		return nil
	}
	return &r
}

func fetchWeek(cfg *config) *weekResp {
	var r weekResp
	if err := apiGet(cfg, "/users/current/stats/last_7_days", &r); err != nil {
		dbg("fetchWeek error: %v", err)
		return nil
	}
	return &r
}

// ─── language inference from heartbeat entities ──────────────────────────────

type heartbeat struct {
	Entity   string  `json:"entity"`
	Time     float64 `json:"time"`
	Language string  `json:"language"`
}

func fetchHeartbeats(cfg *config, date string) []heartbeat {
	var r struct {
		Data []heartbeat `json:"data"`
	}
	if err := apiGet(cfg, "/users/current/heartbeats?date="+date, &r); err != nil {
		dbg("fetchHeartbeats error: %v", err)
		return nil
	}
	return r.Data
}

var extLang = map[string]string{
	".py": "Python", ".go": "Go", ".js": "JavaScript", ".ts": "TypeScript",
	".tsx": "TSX", ".jsx": "JSX", ".rb": "Ruby", ".rs": "Rust",
	".java": "Java", ".kt": "Kotlin", ".swift": "Swift",
	".cpp": "C++", ".cc": "C++", ".cxx": "C++", ".c": "C", ".h": "C/C++",
	".cs": "C#", ".html": "HTML", ".css": "CSS", ".scss": "Sass", ".sass": "Sass",
	".md": "Markdown", ".json": "JSON", ".yaml": "YAML", ".yml": "YAML",
	".sh": "Shell", ".bash": "Shell", ".zsh": "Shell", ".fish": "Shell",
	".vim": "Vim Script", ".lua": "Lua", ".sql": "SQL", ".php": "PHP",
	".dart": "Dart", ".elm": "Elm", ".ex": "Elixir", ".exs": "Elixir",
	".erl": "Erlang", ".hs": "Haskell", ".ml": "OCaml", ".scala": "Scala",
	".clj": "Clojure", ".r": "R", ".jl": "Julia", ".nim": "Nim", ".zig": "Zig",
	".tf": "Terraform", ".tex": "LaTeX", ".toml": "TOML", ".xml": "XML",
	".ipynb": "Jupyter", ".proto": "Protobuf", ".rkt": "Racket",
}

func inferLanguage(entity string) string {
	base := strings.ToLower(filepath.Base(entity))
	switch base {
	case "dockerfile", "containerfile":
		return "Docker"
	case "makefile", "gnumakefile":
		return "Makefile"
	}
	if lang, ok := extLang[strings.ToLower(filepath.Ext(entity))]; ok {
		return lang
	}
	return ""
}

func inferTopLang(cfg *config, date string) (string, int) {
	hbs := fetchHeartbeats(cfg, date)
	if len(hbs) == 0 {
		return "", 0
	}
	counts := map[string]int{}
	for _, h := range hbs {
		if l := inferLanguage(h.Entity); l != "" {
			counts[l]++
		}
	}
	if len(counts) == 0 {
		return "", 0
	}
	best, max := "", 0
	for k, v := range counts {
		if v > max {
			max, best = v, k
		}
	}
	return best, max
}

var (
	cachedHbs []heartbeat
	hbsCached bool
)

func getTodayHeartbeats(cfg *config) []heartbeat {
	if !hbsCached {
		hbsCached = true
		cachedHbs = fetchHeartbeats(cfg, time.Now().Format("2006-01-02"))
	}
	return cachedHbs
}

func getInferredLang(cfg *config) (string, int) {
	hbs := getTodayHeartbeats(cfg)
	if len(hbs) == 0 {
		return "", 0
	}
	counts := map[string]int{}
	for _, h := range hbs {
		if l := inferLanguage(h.Entity); l != "" {
			counts[l]++
		}
	}
	if len(counts) == 0 {
		return "", 0
	}
	best, max := "", 0
	for k, v := range counts {
		if v > max {
			max, best = v, k
		}
	}
	return best, max
}

func countUniqueFilesToday(cfg *config) int {
	hbs := getTodayHeartbeats(cfg)
	seen := map[string]bool{}
	for _, h := range hbs {
		if h.Entity != "" {
			seen[h.Entity] = true
		}
	}
	return len(seen)
}

// ─── streak ──────────────────────────────────────────────────────────────────

type daySummary struct {
	Date    string
	Seconds float64
}

func fetchDailyTotals(cfg *config, daysBack int) []daySummary {
	end := time.Now()
	start := end.AddDate(0, 0, -daysBack)
	var r struct {
		Data []struct {
			GrandTotal struct {
				Seconds float64 `json:"total_seconds"`
			} `json:"grand_total"`
			Range struct {
				Date string `json:"date"`
			} `json:"range"`
		} `json:"data"`
	}
	path := fmt.Sprintf("/users/current/summaries?start=%s&end=%s",
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err := apiGet(cfg, path, &r); err != nil {
		dbg("fetchDailyTotals error: %v", err)
		return nil
	}
	dbg("fetchDailyTotals: got %d days", len(r.Data))
	out := make([]daySummary, len(r.Data))
	for i, d := range r.Data {
		out[i] = daySummary{Date: d.Range.Date, Seconds: d.GrandTotal.Seconds}
	}
	return out
}

// computeStreak counts consecutive active days (>1 min) ending today or yesterday.
// Falls back to "today has activity = streak of 1" if /summaries is unavailable.
func computeStreak(cfg *config) int {
	totals := fetchDailyTotals(cfg, 30)
	active := map[string]bool{}
	for _, t := range totals {
		if t.Seconds > 60 {
			active[t.Date] = true
		}
	}
	if len(active) == 0 {
		dbg("computeStreak: /summaries gave nothing; falling back to today check")
		if t := fetchToday(cfg); t != nil && t.Data.GrandTotal.Seconds > 60 {
			return 1
		}
		return 0
	}
	cur := time.Now()
	// Allow yesterday as the starting anchor if today is empty (day isn't over yet).
	if !active[cur.Format("2006-01-02")] {
		cur = cur.AddDate(0, 0, -1)
		if !active[cur.Format("2006-01-02")] {
			dbg("computeStreak: neither today nor yesterday has activity")
			return 0
		}
	}
	streak := 0
	for active[cur.Format("2006-01-02")] {
		streak++
		cur = cur.AddDate(0, 0, -1)
	}
	dbg("computeStreak: %d days", streak)
	return streak
}

var (
	cachedStreak int
	streakCached bool
)

func getStreak(cfg *config) int {
	if !streakCached {
		streakCached = true
		cachedStreak = computeStreak(cfg)
	}
	return cachedStreak
}

func formatStreak(days int) string {
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func topItem(items []item) string {
	best := ""
	max := 0.0
	for _, x := range items {
		n := strings.TrimSpace(x.Name)
		if n == "" || strings.EqualFold(n, "unknown") || strings.EqualFold(n, "other") {
			continue
		}
		if x.TotalSeconds > max {
			max = x.TotalSeconds
			best = x.Name
		}
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf("%s (%s)", best, fmtDur(max))
}

// ─── system info ─────────────────────────────────────────────────────────────

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
		return "—"
	}
	return filepath.Base(s)
}

func getTerm() string {
	return envOr([]string{"TERM_PROGRAM", "TERM"}, "—")
}

func getEditor() string {
	for _, e := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(e); v != "" {
			return filepath.Base(v)
		}
	}
	return "—"
}

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
		return "—"
	}
	return strings.TrimSuffix(h, ".local")
}

func getUser() string {
	return envOr([]string{"USER", "LOGNAME"}, "—")
}

// ─── setup flow ──────────────────────────────────────────────────────────────

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported os")
	}
	return cmd.Start()
}

const settingsURL = "https://hackatime.hackclub.com/my/wakatime_setup"

func runSetup() (*config, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	fmt.Println("  " + orange + "✦ looking for hackatime…" + reset)
	fmt.Println()

	cliPath := findWakatimeCLI()
	cfgFile, cfgExists := configFileFound()

	if cliPath != "" {
		fmt.Println("  " + green + "✓" + reset + " wakatime-cli: " + dim + cliPath + reset)
	} else {
		fmt.Println("  " + dim + "✗ wakatime-cli not found" + reset)
	}
	if cfgExists {
		fmt.Println("  " + green + "✓" + reset + " config file:  " + dim + cfgFile + reset)
		fmt.Println("  " + dim + "    (file exists but has no api_key — finish setup to add it)" + reset)
	} else {
		fmt.Println("  " + dim + "✗ ~/.wakatime.cfg not found" + reset)
	}
	fmt.Println()
	fmt.Println("  " + txt + "finish your hackatime setup here:" + reset)
	fmt.Println("  " + bold + orange + settingsURL + reset)
	fmt.Println()
	fmt.Println("  " + dim + "→ the page walks you through installing wakatime-cli and writing ~/.wakatime.cfg." + reset)
	fmt.Println("  " + dim + "→ once done, your editor plugins " + reset + bold + "and" + reset + dim + " hackfetch will both work." + reset)
	fmt.Println()
	fmt.Print("  " + orange + "open it in your browser? " + dim + "[Y/n] " + reset)

	resp, _ := reader.ReadString('\n')
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp == "" || resp == "y" || resp == "yes" {
		if err := openBrowser(settingsURL); err != nil {
			fmt.Println("  " + dim + "(browser open failed — copy the url manually)" + reset)
		}
	}

	fmt.Println()
	fmt.Print("  " + orange + "press " + bold + "[enter]" + reset + orange + " when setup is done, or " + bold + "q" + reset + orange + " to quit: " + reset)
	resp2, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(resp2)) == "q" {
		return nil, fmt.Errorf("canceled")
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("still no api_key in ~/.wakatime.cfg — finish the setup page first, then re-run hackfetch")
	}
	fmt.Println()
	fmt.Println("  " + green + "✓ found your key in ~/.wakatime.cfg" + reset)
	fmt.Println()
	return cfg, nil
}

// ─── render ──────────────────────────────────────────────────────────────────

type field struct {
	label, value string
}

func padRunes(s string, w int) string {
	n := utf8.RuneCountInString(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

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

// ─── main ────────────────────────────────────────────────────────────────────

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// buildFields constructs the field list shown next to the logo.
// Shared by render and exportSVG so they stay in sync.
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
		if u := fetchUser(cfg); u != nil {
			gotAny = true
			fields = append(fields, field{"slack", "@" + u.Username})
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
		if t := fetchToday(cfg); t != nil {
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
		if w := fetchWeek(cfg); w != nil {
			gotAny = true
			fields = append(fields, field{"7-day total", fmtDur(w.Data.TotalSeconds)})
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
			if len(w.Data.Machines) > 1 {
				fields = append(fields, field{"machines", fmt.Sprintf("%d", len(w.Data.Machines))})
			}
		}
		if !gotAny && lastAPIErr != nil {
			msg := lastAPIErr.Error()
			if strings.Contains(msg, "401") {
				msg = "auth failed — rotate key at hackatime.hackclub.com/my/settings/privacy or run `hackfetch -setup`"
			}
			fields = append(fields, field{"⚠ api", dim + msg + reset})
		}
	} else if noNet {
		fields = append(fields, field{"hackatime", dim + "offline (-no-net)" + reset})
	}

	return fields
}

// resetCaches clears memoized API data so a watch-mode loop fetches fresh.
func resetCaches() {
	hbsCached = false
	cachedHbs = nil
	streakCached = false
	cachedStreak = 0
	lastAPIErr = nil
}

// runWatch re-renders the fetch on an interval until interrupted.
func runWatch(logoLines []string, sch scheme, cfg *config, noNet, verbose bool, interval time.Duration) {
	for {
		resetCaches()
		fmt.Print("\x1b[2J\x1b[H")
		fields := buildFields(cfg, noNet, verbose)
		render(logoLines, sch, fields)
		fmt.Printf("  %sat %s · refreshes every %ds · ctrl-c to quit%s\n\n",
			dim, time.Now().Format("15:04:05"), int(interval.Seconds()), reset)
		time.Sleep(interval)
	}
}

// ─── svg export ──────────────────────────────────────────────────────────────

func stripANSI(s string) string {
	var b strings.Builder
	n := len(s)
	for i := 0; i < n; i++ {
		if s[i] == 0x1b && i+1 < n && s[i+1] == '[' {
			for i < n && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// ansi256ToHex maps an ANSI 256-color code to a hex string for SVG fills.
func ansi256ToHex(code int) string {
	if code < 0 || code > 255 {
		return "#ececec"
	}
	if code < 16 {
		std := []string{
			"#000000", "#cc0000", "#4e9a06", "#c4a000",
			"#3465a4", "#75507b", "#06989a", "#d3d7cf",
			"#555753", "#ef2929", "#8ae234", "#fce94f",
			"#729fcf", "#ad7fa8", "#34e2e2", "#eeeeec",
		}
		return std[code]
	}
	if code < 232 {
		n := code - 16
		r := n / 36
		g := (n / 6) % 6
		b := n % 6
		v := func(x int) int {
			if x == 0 {
				return 0
			}
			return 55 + x*40
		}
		return fmt.Sprintf("#%02x%02x%02x", v(r), v(g), v(b))
	}
	v := 8 + (code-232)*10
	return fmt.Sprintf("#%02x%02x%02x", v, v, v)
}

func svgLineColor(sch scheme, lineIdx int) string {
	if len(sch.colors) == 0 {
		return "#ff8c3a"
	}
	return ansi256ToHex(sch.colors[lineIdx%len(sch.colors)])
}

// svgLogoLine renders a single logo line as SVG tspans honoring the scheme mode.
func svgLogoLine(sch scheme, line string, lineIdx int) string {
	if sch.mode != modePerChar {
		return fmt.Sprintf(`<tspan fill="%s">%s</tspan>`,
			svgLineColor(sch, lineIdx), escapeXML(line))
	}
	var b strings.Builder
	i := 0
	for _, r := range line {
		c := ansi256ToHex(sch.colors[(i+lineIdx)%len(sch.colors)])
		fmt.Fprintf(&b, `<tspan fill="%s">%s</tspan>`, c, escapeXML(string(r)))
		i++
	}
	return b.String()
}

// exportSVG writes a shareable card of the fetch as an SVG file.
func exportSVG(path string, logoLines []string, sch scheme, fields []field) error {
	const (
		charW   = 9
		lineH   = 20
		padding = 28
		logoW   = 33
		sepW    = 3
		fieldW  = 50
	)

	rows := len(logoLines)
	if len(fields) > rows {
		rows = len(fields)
	}
	if rows < 5 {
		rows = 5
	}

	width := padding*2 + (logoW+sepW+fieldW)*charW
	height := padding*2 + (rows+2)*lineH

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="ui-monospace, SFMono-Regular, Menlo, Monaco, monospace" font-size="14">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" rx="16" ry="16" fill="#0a0a0c"/>`)

	for i := 0; i < rows; i++ {
		y := padding + (i+1)*lineH

		if i < len(logoLines) {
			fmt.Fprintf(&b,
				`<text x="%d" y="%d" xml:space="preserve">%s</text>`,
				padding, y, svgLogoLine(sch, logoLines[i], i))
		}

		if i < len(fields) {
			f := fields[i]
			x := padding + (logoW+sepW)*charW
			labelClean := stripANSI(f.label)
			valueClean := stripANSI(f.value)
			if labelClean == "" {
				fmt.Fprintf(&b,
					`<text x="%d" y="%d" fill="#ececec" xml:space="preserve">%s</text>`,
					x, y, escapeXML(valueClean))
			} else {
				fmt.Fprintf(&b,
					`<text x="%d" y="%d" fill="%s" xml:space="preserve">%s</text>`,
					x, y, svgLineColor(sch, i), escapeXML(labelClean))
				fmt.Fprintf(&b,
					`<text x="%d" y="%d" fill="#ececec" xml:space="preserve">%s</text>`,
					x+13*charW, y, escapeXML(valueClean))
			}
		}
	}

	footerY := padding + (rows+1)*lineH + 8
	fmt.Fprintf(&b,
		`<text x="%d" y="%d" fill="#757580" font-size="11" xml:space="preserve">hackfetch · github.com/xerneas3318/hackfetch</text>`,
		padding, footerY)

	b.WriteString(`</svg>`)

	return os.WriteFile(path, []byte(b.String()), 0644)
}

func main() {
	loadCustomSchemes()

	defaultLogo := envOr([]string{"HACKFETCH_LOGO"}, "hackclub")
	defaultColor := envOr([]string{"HACKFETCH_COLOR"}, "hackclub")

	logoFlag := flag.String("logo", defaultLogo, "logo name (see -list)")
	colorFlag := flag.String("color", defaultColor, "color scheme (see -list)")
	setupFlag := flag.Bool("setup", false, "re-run the api key setup flow")
	listFlag := flag.Bool("list", false, "list available logos and color schemes")
	noNet := flag.Bool("no-net", false, "skip api calls (offline mode)")
	defaultVerbose := os.Getenv("HACKFETCH_VERBOSE") != ""
	verboseFlag := flag.Bool("v", defaultVerbose, "verbose: also show editor + category breakdowns (set HACKFETCH_VERBOSE=1 to make default)")
	watchFlag := flag.Bool("watch", false, "live mode: refresh every 30s until ctrl+c")
	exportFlag := flag.String("export", "", "export the fetch as an SVG card to a file (e.g. card.svg)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "hackfetch — hack club system fetch")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "usage:")
		fmt.Fprintln(os.Stderr, "  hackfetch [logo] [color] [flags]")
		fmt.Fprintln(os.Stderr, "  hackfetch logo <name> color <name> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "examples:")
		fmt.Fprintln(os.Stderr, "  hackfetch                              # defaults")
		fmt.Fprintln(os.Stderr, "  hackfetch stardance rainbow            # shorthand")
		fmt.Fprintln(os.Stderr, "  hackfetch logo flag color pride        # keyword form")
		fmt.Fprintln(os.Stderr, "  hackfetch -logo orpheus -color ocean   # flag form")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "logos:  "+strings.Join(sortedKeys(logos), ", "))
		fmt.Fprintln(os.Stderr, "colors: "+strings.Join(sortedKeys(schemes), ", "))
	}
	flag.Parse()

	// Allow positional args:
	//   hackfetch stardance rainbow
	//   hackfetch logo stardance color rainbow
	extras := flag.Args()
	for i := 0; i < len(extras); i++ {
		a := extras[i]
		switch a {
		case "logo":
			if i+1 < len(extras) {
				*logoFlag = extras[i+1]
				i++
			}
		case "color":
			if i+1 < len(extras) {
				*colorFlag = extras[i+1]
				i++
			}
		case "help", "h":
			flag.Usage()
			return
		default:
			if _, ok := logos[a]; ok {
				*logoFlag = a
			} else if _, ok := schemes[a]; ok {
				*colorFlag = a
			} else {
				fmt.Fprintf(os.Stderr, "unknown arg %q (try -h)\n", a)
			}
		}
	}

	if *listFlag {
		fmt.Println("logos:")
		for _, k := range sortedKeys(logos) {
			fmt.Println("  " + k)
		}
		fmt.Println("\ncolors:")
		for _, k := range sortedKeys(schemes) {
			fmt.Println("  " + k)
		}
		return
	}

	logoLines, ok := logos[*logoFlag]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown logo %q (try -list)\n", *logoFlag)
		logoLines = logos["hackclub"]
	}
	if *colorFlag == "auto" {
		*colorFlag = resolveAutoScheme()
	}
	sch, ok := schemes[*colorFlag]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown color %q (try -list)\n", *colorFlag)
		sch = schemes["hackclub"]
	}

	// Force login: refuse to render without a hackatime key, unless -no-net.
	var cfg *config
	if !*noNet {
		c, _ := loadConfig()
		if *setupFlag {
			c = nil
		}
		for c == nil {
			nc, err := runSetup()
			if err != nil {
				fmt.Fprintln(os.Stderr, "  "+dim+"setup failed: "+err.Error()+reset)
				fmt.Println()
				os.Exit(1)
			}
			c = nc
		}
		cfg = c
	}

	if *watchFlag {
		runWatch(logoLines, sch, cfg, *noNet, *verboseFlag, 30*time.Second)
		return
	}

	fields := buildFields(cfg, *noNet, *verboseFlag)

	if *exportFlag != "" {
		if err := exportSVG(*exportFlag, logoLines, sch, fields); err != nil {
			fmt.Fprintf(os.Stderr, "export failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ wrote %s\n", *exportFlag)
		return
	}

	render(logoLines, sch, fields)
}
