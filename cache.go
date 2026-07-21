package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// disk cache for hackatime stuff
// makes repeat runs instant and keeps tmux polling from being obnoxious
// watch mode ignores it obv

type diskCache struct {
	SavedAt      time.Time   `json:"saved_at"`
	APIURL       string      `json:"api_url,omitempty"`
	User         *userInfo   `json:"user,omitempty"`
	Today        *todayResp  `json:"today,omitempty"`
	Week         *weekResp   `json:"week,omitempty"`
	Streak       int         `json:"streak"`
	Heartbeats   []heartbeat `json:"heartbeats,omitempty"`
	HeartbeatsAt time.Time   `json:"heartbeats_at,omitempty"`

	Spans    []span `json:"spans,omitempty"`
	SpansUID string `json:"spans_uid,omitempty"`
}

func cacheFilePath() string {
	if v := os.Getenv("HACKFETCH_CACHE"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "hackfetch", "last.json")
}


// ttl 0 = skip cache
// HACKFETCH_CACHE_TTL overrides
func cacheTTL(def time.Duration) time.Duration {
	if v := os.Getenv("HACKFETCH_CACHE_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return time.Duration(n) * time.Second
		}
	}
	return def
}

// hit = loaded from disk skipped the api
func loadCacheIfFresh(ttl time.Duration) bool {
	if ttl <= 0 {
		return false
	}
	p := cacheFilePath()
	if p == "" {
		return false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}

	var dc diskCache
	if err := json.Unmarshal(data, &dc); err != nil {
		dbg("cache parse error %v", err)
		return false
	}
	if time.Since(dc.SavedAt) > ttl {
		dbg("cache stale %s", time.Since(dc.SavedAt).Round(time.Second))
		return false
	}

	// slam the process caches
	cachedUser = dc.User
	userCached = true
	cachedTodayVal = dc.Today
	todayCached = true
	cachedWeekVal = dc.Week
	weekCached = true
	cachedStreak = dc.Streak
	streakCached = true

	if len(dc.Heartbeats) > 0 && time.Since(dc.HeartbeatsAt) < 24*time.Hour {
		cachedHbs = dc.Heartbeats
		hbsCached = true
	}
	if len(dc.Spans) > 0 && dc.SpansUID != "" {
		cachedSpans = dc.Spans
		cachedSpansUID = dc.SpansUID
		spansCached = true
	}

	dbg("cache hit age %s", time.Since(dc.SavedAt).Round(time.Second))
	return true
}

// best effort write
// silent on fail so a read only home doesnt nuke the fetch
// TODO prune old cache files someday if per user caches ever happen
// TODO this is kinda slop for now good enough
func saveCacheFromMemory(cfg *config) {
	p := cacheFilePath()
	if p == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		dbg("cache mkdir failed %v", err)
		return
	}

	dc := diskCache{
		SavedAt: time.Now(),
		User:    cachedUser,
		Today:   cachedTodayVal,
		Week:    cachedWeekVal,
		Streak:  cachedStreak,
	}
	if cfg != nil {
		dc.APIURL = cfg.APIURL
	}

	if hbsCached {
		dc.Heartbeats = cachedHbs
		dc.HeartbeatsAt = time.Now()
	}
	if spansCached {
		dc.Spans = cachedSpans
		dc.SpansUID = cachedSpansUID
	}

	data, err := json.Marshal(dc)
	if err != nil {
		dbg("cache marshal failed %v", err)
		return
	}


	// atomic write tmp then rename
	// otherwise a reader could catch a half written file and cry
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		dbg("cache write failed %v", err)
		return
	}
	if err := os.Rename(tmp, p); err != nil {
		dbg("cache rename failed %v", err)
		os.Remove(tmp)
	}
}
