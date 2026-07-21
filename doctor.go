package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// checklist that yells at the user if something is broken
// green = fine yellow = eh red = you did something wrong
// exit 1 on any red so scripts can gate on it
// TODO add a --json mode someday for tooling
func runDoctor() int {
	fmt.Println()
	fmt.Printf("  %s✦ hackfetch doctor%s\n", orange, reset)
	fmt.Println()

	fail := 0
	warn := 0

	// config file
	if p, ok := configFileFound(); ok {
		pass("wakatime config exists", p)
	} else {
		pathHint := "(no ~/.wakatime.cfg or equivalent)"
		if p != "" {
			pathHint = p
		}
		miss("no wakatime config",
			"run "+bold+"hackfetch -setup"+reset+dim+" to create one at "+pathHint+reset)
		fail++
	}

	// api key
	cfg, err := loadConfig()
	if err != nil || cfg == nil || cfg.APIKey == "" {
		miss("no api_key in config",
			"finish the setup at "+bold+"hackatime.hackclub.com/my/wakatime_setup"+reset)
		fail++
	} else {
		pass("api_key present", "loaded from "+cfg.APIURL)
	}

	// wakatime-cli optional but nice
	if cliPath := findWakatimeCLI(); cliPath != "" {
		pass("wakatime-cli found", cliPath)
	} else {
		warnLine("wakatime-cli not on PATH",
			"editor plugins wont send heartbeats until you install it")
		warn++
	}


	// smoke test the api using the cheapest endpoint that returns a stable shape
	if cfg != nil && cfg.APIKey != "" {
		start := time.Now()
		if t := fetchToday(cfg); t != nil {
			ms := time.Since(start).Milliseconds()
			pass("hackatime api reachable",
				fmt.Sprintf("today %s (%dms)", fmtDur(t.Data.GrandTotal.Seconds), ms))
		} else {
			hint := "check network + api key"
			if lastAPIErr != nil && strings.Contains(lastAPIErr.Error(), "401") {
				hint = "auth failed rotate key at hackatime.hackclub.com/my/settings/privacy"
			} else if lastAPIErr != nil {
				hint = lastAPIErr.Error()
			}
			miss("hackatime api call failed", hint)
			fail++
		}
	}

	// terminal color depth
	if useTrueColor() {
		pass("terminal supports truecolor", "$COLORTERM="+os.Getenv("COLORTERM"))
	} else {
		warnLine("terminal is 256-color only",
			"gradients will be quantized set COLORTERM=truecolor if your terminal actually supports it")
		warn++
	}

	// separate smoke test for dns / tls so a broken cert shows up distinct from auth
	if cfg != nil {
		host := strings.TrimPrefix(strings.TrimPrefix(cfg.APIURL, "https://"), "http://")
		if i := strings.Index(host, "/"); i >= 0 {
			host = host[:i]
		}
		if _, err := (&http.Client{Timeout: 3 * time.Second}).Get("https://" + host); err != nil {
			warnLine("could not reach "+host,
				"dns or tls issue "+err.Error())
			warn++
		}
	}

	// xdg-open / open / rundll32 for -setup
	if opener, ok := browserOpener(); ok {
		pass("browser opener available", opener)
	} else {
		warnLine("no browser opener for this platform",
			"-setup will still print the URL but wont auto-open it")
		warn++
	}

	fmt.Println()
	switch {
	case fail == 0 && warn == 0:
		fmt.Printf("  %s✓ everything looks good%s\n", green, reset)
	case fail == 0:
		fmt.Printf("  %s✓ working with %d warning(s)%s\n", green, warn, reset)
	default:
		fmt.Printf("  %s✗ %d problem(s) %d warning(s)%s\n", orange, fail, warn, reset)
	}
	fmt.Println()

	if fail > 0 {
		return 1
	}
	return 0
}

// little status line helpers
// fixed width so the doctor lines up
func pass(label, detail string) {
	fmt.Printf("  %s✓%s %-32s %s%s%s\n", green, reset, label, dim, detail, reset)
}

func warnLine(label, detail string) {
	fmt.Printf("  %s!%s %-32s %s%s%s\n", orange, reset, label, dim, detail, reset)
}

func miss(label, detail string) {
	fmt.Printf("  %s✗%s %-32s %s%s%s\n", orange, reset, label, dim, detail, reset)
}

// what command opens a url on this os
// doctor reports it without actually launching anything
func browserOpener() (string, bool) {
	var name string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "linux":
		name = "xdg-open"
	case "windows":
		return "rundll32", true
	default:
		return "", false
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, true
	}
	return "", false
}
