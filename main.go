package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// bumped on release. also shown in -json output so scripts can tell what shape they got
const version = "2.3.0"

// HACKFETCH_DEBUG=1 turns on the noisy logs
var debug = os.Getenv("HACKFETCH_DEBUG") != ""

func dbg(format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "[hackfetch] "+format+"\n", args...)
	}
}

// last day of stardance
// after this the countdown field hides itself
const stardanceEnd = "2026-09-30"

// days remaining and whether stardance is still on
// second return false = hide the field
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

// stardust count from HACKFETCH_STARDUST
// TODO auto sync from the stardance page someday would be sick
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
	verboseFlag := flag.Bool("v", defaultVerbose, "verbose: also show slack, 7-day chart, machines, editor, category (set HACKFETCH_VERBOSE=1 to make default)")
	watchFlag := flag.Bool("watch", false, "live mode: refresh every 30s until ctrl+c")
	exportFlag := flag.String("export", "", "export the fetch as an image (e.g. card.png, card.jpg, card.svg)")
	statusFlag := flag.Bool("status", false, "print a one-line summary for status bars (tmux, lualine) and exit")
	sparklineFlag := flag.Bool("sparkline", false, "with -status, append a 7-day sparkline of coding time")
	doctorFlag := flag.Bool("doctor", false, "diagnose your setup (config, api key, network, terminal) and exit")
	noCache := flag.Bool("no-cache", false, "skip the on-disk cache and force a fresh api fetch")
	leaderboardFlag := flag.String("leaderboard", "", "show hackatime leaderboard: daily or weekly")
	jsonFlag := flag.Bool("json", false, "dump the fetched data as JSON to stdout and exit")
	vsFlag := flag.String("vs", "", "head to head compare against another hackatime user (accepts @name, name, or a hackatime profile url)")
	readmeFlag := flag.Bool("readme", false, "print a markdown stats card ready to paste into your github profile README")
	recordsFlag := flag.Bool("records", false, "show your all time personal records: biggest day, longest streak ever, longest session, days coded, lifetime hours")
	versionFlag := flag.Bool("version", false, "print version and exit")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "hackfetch: hack club system fetch")
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
		fmt.Fprintln(os.Stderr, "  hackfetch vs @speedhawks               # head to head compare")
		fmt.Fprintln(os.Stderr, "  hackfetch -records                     # your all time personal records")
		fmt.Fprintln(os.Stderr, "  hackfetch -readme > stats.md           # markdown for github profile")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "logos:  "+strings.Join(sortedKeys(logos), ", "))
		fmt.Fprintln(os.Stderr, "colors: "+strings.Join(sortedKeys(schemes), ", "))
	}
	flag.Parse()

	if *versionFlag {
		fmt.Println("hackfetch v" + version)
		return
	}

	// positional args
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
		case "version":
			fmt.Println("hackfetch v" + version)
			return
		case "vs":
			if i+1 < len(extras) {
				*vsFlag = extras[i+1]
				i++
			}
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

	// status mode is fully non interactive
	// never opens setup never blocks never spams errors
	// tmux/lualine can call this every N seconds without deadlocking
	if *statusFlag {
		if *noNet {
			return
		}
		cfg, _ := loadConfig()
		if cfg == nil {
			return
		}
		// tmux polls at ~60s so a 60s cache basically zeros out the network cost
		hit := false
		if !*noCache {
			hit = loadCacheIfFresh(cacheTTL(60 * time.Second))
		}
		if !hit {
			prefetchAll(cfg)
			saveCacheFromMemory(cfg)
		}
		printStatus(cfg, *sparklineFlag)
		return
	}

	// doctor mode: run the checklist and bounce
	// non zero exit code on any red so scripts can gate on it
	if *doctorFlag {
		os.Exit(runDoctor())
	}

	// records mode: crunch our full spans history for lifetime records
	// biggest day longest streak ever longest session etc all from data we already fetch
	if *recordsFlag {
		if *noNet {
			fmt.Fprintln(os.Stderr, "records mode needs the network. drop -no-net.")
			os.Exit(1)
		}
		cfg, _ := loadConfig()
		if cfg == nil {
			fmt.Fprintln(os.Stderr, "run "+bold+"hackfetch -setup"+reset+" first")
			os.Exit(1)
		}
		printRecords(cfg)
		return
	}

	// vs mode: fetch our stats + theirs and print side by side
	// their spans come from the public /users/:username/heartbeats/spans endpoint
	if *vsFlag != "" {
		if *noNet {
			fmt.Fprintln(os.Stderr, "vs mode needs the network. drop -no-net.")
			os.Exit(1)
		}
		cfg, _ := loadConfig()
		if cfg == nil {
			fmt.Fprintln(os.Stderr, "run "+bold+"hackfetch -setup"+reset+" first")
			os.Exit(1)
		}
		printVs(cfg, *vsFlag)
		return
	}

	// leaderboard mode: public endpoint no auth needed
	// still needs cfg for the api URL + optional self-highlight
	if *leaderboardFlag != "" {
		if *noNet {
			return
		}
		cfg, _ := loadConfig()
		if cfg == nil {
			// no config = no self highlight but the leaderboard itself is public
			// hackfetch still needs a base URL though so bail cleanly
			fmt.Fprintln(os.Stderr, "run "+bold+"hackfetch -setup"+reset+" first")
			os.Exit(1)
		}
		// piggyback fetchUser so we can highlight the current user in the list
		go func() { cachedFetchUser(cfg) }()
		printLeaderboard(cfg, *leaderboardFlag, 10)
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

	// refuse to render without a hackatime key unless -no-net
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

	// one-shot uses a shorter TTL than -status
	// a re-run within ~30s (e.g. between two devlog attempts) is instant
	// but stats still stay reasonably fresh
	cacheHit := false
	if cfg != nil && !*noNet && !*noCache {
		cacheHit = loadCacheIfFresh(cacheTTL(30 * time.Second))
	}
	var stopSpinner func()
	if cfg != nil && !*noNet && !cacheHit {
		stopSpinner = startSpinner("fetching Hackatime...")
		prefetchAll(cfg)
		saveCacheFromMemory(cfg)
	}
	fields := buildFields(cfg, *noNet, *verboseFlag)
	if stopSpinner != nil {
		stopSpinner()
	}

	// json dump path shares the same cached data buildFields just consumed
	// no color codes no ansi just clean json to stdout
	if *jsonFlag {
		printJSON(cfg, *noNet)
		return
	}

	// markdown card for github profile readmes
	// uses the same cached data as the normal render so no extra api calls
	if *readmeFlag {
		printReadmeMarkdown(cfg, *noNet)
		return
	}

	if *exportFlag != "" {
		var err error
		switch strings.ToLower(filepath.Ext(*exportFlag)) {
		case ".svg":
			err = exportSVG(*exportFlag, logoLines, sch, fields)
		case ".png":
			err = exportRaster(*exportFlag, logoLines, sch, fields, "png")
		case ".jpg", ".jpeg":
			err = exportRaster(*exportFlag, logoLines, sch, fields, "jpeg")
		default:
			fmt.Fprintf(os.Stderr, "unsupported export extension %q (use .png, .jpg, or .svg)\n", filepath.Ext(*exportFlag))
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "export failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ wrote %s\n", *exportFlag)
		return
	}

	render(logoLines, sch, fields)
}
