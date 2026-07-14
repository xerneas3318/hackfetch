package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// base url for the native hackatime api (not wakatime-compat)
func (c *config) nativeBase() string {
	if i := strings.Index(c.APIURL, "/api/hackatime/"); i >= 0 {
		return c.APIURL[:i] + "/api/v1"
	}
	return c.APIURL
}

// last transport error so a totally dead api shows one friendly hint
var lastAPIErr error


// shared http client
// keeping tls connections alive matters a ton for the parallel prefetch
// without this every call did a fresh handshake and it was slow af
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        16,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     30 * time.Second,
	},
}

func apiGetURL(cfg *config, url string, into any) error {
	dbg("GET %s", url)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(cfg.APIKey, "waka_"))
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		dbg("transport error %v", err)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dbg("status=%d body=%.300s", resp.StatusCode, string(body))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, into)
}

func apiGet(cfg *config, path string, into any) error {
	return apiGetURL(cfg, cfg.APIURL+path, into)
}

type userInfo struct {
	Username string
}

// grabs the slack uid from the already cached week response
// used to be its own round trip until i realized week returns it too
// saves one api call per fetch which adds up
func fetchUser(cfg *config) *userInfo {
	w := cachedFetchWeek(cfg)
	if w == nil {
		dbg("fetchUser no cached week response yet")
		return nil
	}
	if w.Data.Username != "" {
		return &userInfo{Username: w.Data.Username}
	}
	if w.Data.UserID != "" {
		return &userInfo{Username: w.Data.UserID}
	}
	dbg("fetchUser username missing from week response")
	return nil
}

// walks any json for the first non empty match
// takes strings AND numbers because hackatime returns user_id as both depending on the endpoint (lol)
func findStringField(v any, keys []string) string {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range keys {
			switch val := t[k].(type) {
			case string:
				if val != "" {
					return val
				}
			case float64:
				if val != 0 {
					return fmt.Sprintf("%d", int64(val))
				}
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
		Username           string  `json:"username"`
		UserID             string  `json:"user_id"`
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
		dbg("fetchToday error %v", err)
		return nil
	}
	return &r
}


// in-memory caches
// resetCaches in render.go clears these between watch refreshes
var (
	cachedUser     *userInfo
	userCached     bool
	cachedTodayVal *todayResp
	todayCached    bool
	cachedWeekVal  *weekResp
	weekCached     bool
)

func cachedFetchUser(cfg *config) *userInfo {
	if !userCached {
		userCached = true
		cachedUser = fetchUser(cfg)
	}
	return cachedUser
}

func cachedFetchToday(cfg *config) *todayResp {
	if !todayCached {
		todayCached = true
		cachedTodayVal = fetchToday(cfg)
	}
	return cachedTodayVal
}

func cachedFetchWeek(cfg *config) *weekResp {
	if !weekCached {
		weekCached = true
		cachedWeekVal = fetchWeek(cfg)
	}
	return cachedWeekVal
}

// fires the independent hackatime calls at the same time
// buildFields runs after and reads whatever landed in the caches
// week -> user -> streak has to be sequential since streak needs the slack uid
// today runs fully in parallel
// heartbeats stays lazy nobody needs them unless top language is unknown
func prefetchAll(cfg *config) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		cachedFetchWeek(cfg)
		cachedFetchUser(cfg)
		getStreak(cfg)
	}()
	go func() { defer wg.Done(); cachedFetchToday(cfg) }()
	wg.Wait()
}

func fetchWeek(cfg *config) *weekResp {
	var r weekResp
	if err := apiGet(cfg, "/users/current/stats/last_7_days", &r); err != nil {
		dbg("fetchWeek error %v", err)
		return nil
	}
	return &r
}

type heartbeat struct {
	Entity   string  `json:"entity"`
	Time     float64 `json:"time"`
	Language string  `json:"language"`
}

// short circuit after the first 404
// the wakatime compat heartbeats?date= endpoint is gone on the current deployment
// no point wasting more round trips this run
var heartbeatsEndpointDead bool

func fetchHeartbeats(cfg *config, date string) []heartbeat {
	if heartbeatsEndpointDead {
		return nil
	}
	var r struct {
		Data []heartbeat `json:"data"`
	}
	if err := apiGet(cfg, "/users/current/heartbeats?date="+date, &r); err != nil {
		dbg("fetchHeartbeats error %v", err)
		if strings.Contains(err.Error(), "HTTP 404") {
			heartbeatsEndpointDead = true
		}
		return nil
	}
	return r.Data
}

// fallback file extension -> language table
// only used when hackatime says the top language is unknown
// TODO more extensions someday probably solidity zig etc
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

// one fetch per run
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


// spans = every coding session in the users entire history
// one call ~150KB gives us everything for streak and the sparkline
// hackatimes /users/current/summaries endpoint is gone so this is how we do it now
type span struct {
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
	Duration  float64 `json:"duration"`
}

// share the same spans call across streak + sparkline so we only pay for it once
var (
	cachedSpans    []span
	spansCached    bool
	cachedSpansUID string
)

func fetchSpans(cfg *config, slackUID string) []span {
	if slackUID == "" {
		return nil
	}
	if spansCached && cachedSpansUID == slackUID {
		return cachedSpans
	}
	var r struct {
		Spans []span `json:"spans"`
	}
	url := cfg.nativeBase() + "/users/" + slackUID + "/heartbeats/spans"
	if err := apiGetURL(cfg, url, &r); err != nil {
		dbg("fetchSpans error %v", err)
		return nil
	}
	dbg("fetchSpans got %d spans", len(r.Spans))
	cachedSpans = r.Spans
	spansCached = true
	cachedSpansUID = slackUID
	return r.Spans
}

// unique local dates that had any activity
func activeDaysFromSpans(spans []span) map[string]bool {
	out := map[string]bool{}
	for _, s := range spans {
		if s.Duration <= 0 {
			continue
		}
		d := time.Unix(int64(s.StartTime), 0).Format("2006-01-02")
		out[d] = true
	}
	return out
}

// consecutive active days ending today or yesterday
// yesterday can anchor if today hasnt started yet
func computeStreak(cfg *config) int {
	u := cachedFetchUser(cfg)
	if u == nil || u.Username == "" {
		dbg("computeStreak no user info hiding streak")
		return 0
	}
	spans := fetchSpans(cfg, u.Username)
	if len(spans) == 0 {
		return 0
	}
	active := activeDaysFromSpans(spans)
	cur := time.Now()
	if !active[cur.Format("2006-01-02")] {
		cur = cur.AddDate(0, 0, -1)
		if !active[cur.Format("2006-01-02")] {
			dbg("computeStreak neither today nor yesterday has activity")
			return 0
		}
	}
	streak := 0
	for active[cur.Format("2006-01-02")] {
		streak++
		cur = cur.AddDate(0, 0, -1)
	}
	dbg("computeStreak %d days", streak)
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

// most used item name + its duration
// skips the unknown/other buckets hackatime returns when it cant figure out attribution
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
