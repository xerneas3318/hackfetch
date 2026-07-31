package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hackatime creds loaded from ~/.wakatime.cfg
type config struct {
	APIKey string
	APIURL string
}

// first existing wakatime config path
// or the default write location if none of the known ones exist yet
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

// path to wakatime-cli if installed else ""
// doctor + setup use this to say whats already there
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

// path + whether it exists on disk
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

// parses the wakatime config file
// returns the api key + url
// forgiving about inline comments and surrounding quotes since some editors write those
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
		k := strings.ToLower(strings.TrimSpace(parts[0]))
		v := cleanConfigValue(parts[1])
		switch k {
		case "api_key", "api-key", "apikey":
			cfg.APIKey = v
		case "api_url", "api-url", "apiurl":
			cfg.APIURL = strings.TrimSuffix(v, "/")
		}
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("no api_key in ~/.wakatime.cfg")
	}
	return cfg, nil
}

// strips whitespace, inline comments (# or ;), and surrounding single or double quotes
// wakatime configs written by hand or third party tools sometimes have any of these
func cleanConfigValue(raw string) string {
	v := raw
	// drop inline comments but only if the comment marker is preceded by whitespace
	// (a # inside a quoted key would be weird but lets not misparse)
	for _, m := range []string{" #", "\t#", " ;", "\t;"} {
		if i := strings.Index(v, m); i >= 0 {
			v = v[:i]
		}
	}
	v = strings.TrimSpace(v)
	// strip surrounding matched quotes
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return v
}

// scans the config file for an api_key line without caring about its value
// setup uses this to distinguish "file is missing the key entirely" from
// "key line is present but value looks off" so we can message the user accurately
func configHasAPIKeyLine() bool {
	path, err := cfgPath()
	if err != nil {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
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
		k := strings.ToLower(strings.TrimSpace(parts[0]))
		if k == "api_key" || k == "api-key" || k == "apikey" {
			return true
		}
	}
	return false
}

// writes or replaces the api_key line in the wakatime config
// keeps any other settings that were already in the file
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
