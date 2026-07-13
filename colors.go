package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ansi escapes used everywhere
const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	white  = "\x1b[38;5;231m"
	dim    = "\x1b[38;5;243m"
	txt    = "\x1b[38;5;253m"
	orange = "\x1b[38;5;208m"
	green  = "\x1b[38;5;42m"
)

// 256-color foreground escape
func ansi(code int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", code)
}

// 24-bit foreground escape
func ansiRGB(r, g, b uint8) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// does this terminal do truecolor
// HACKFETCH_TRUECOLOR=1 / =0 forces it either way
// otherwise trust COLORTERM (iterm2 kitty wezterm alacritty modern gnome-terminal etc all set this)
func useTrueColor() bool {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("HACKFETCH_TRUECOLOR"))); v != "" {
		return v == "1" || v == "true" || v == "yes" || v == "on"
	}
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	return ct == "truecolor" || ct == "24bit"
}

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


// TODO more pride flags someday nb ace etc
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
	"bi":        {[]int{199, 99, 27}, modePerLine},
	"bisexual":  {[]int{199, 99, 27}, modePerLine},
	"pan":       {[]int{199, 226, 33}, modePerLine},
	"pansexual": {[]int{199, 226, 33}, modePerLine},
}

// wraps a string in the ansi codes for the scheme
// truecolor path smoothly interpolates per-char gradients across the line
// falls back to the classic 256-color cycle on older terminals
func colorize(s string, sch scheme, lineIdx int) string {
	if sch.mode == modeSingle {
		if useTrueColor() {
			r, g, b := ansi256ToRGB(sch.colors[0])
			return ansiRGB(r, g, b) + s + reset
		}
		return ansi(sch.colors[0]) + s + reset
	}

	tc := useTrueColor()

	if sch.mode == modePerLine {
		idx := sch.colors[lineIdx%len(sch.colors)]
		if tc {
			r, g, b := ansi256ToRGB(idx)
			return ansiRGB(r, g, b) + s + reset
		}
		return ansi(idx) + s + reset
	}

	// per-char fallback (no truecolor)
	if !tc {
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

	// truecolor per-char
	// spread the palette across the line as a cyclic gradient
	// lineIdx nudges the start so rows stagger like a diagonal wave (keeps the vibe of the old cycle mode)
	runes := []rune(s)
	L := len(runes)
	if L == 0 {
		return ""
	}
	if L == 1 {
		r, g, b := paletteRGBWrap(sch.colors, 0)
		return ansiRGB(r, g, b) + string(runes[0]) + reset
	}
	offset := float64(lineIdx) / float64(len(sch.colors)*3)
	var buf strings.Builder
	for i, ch := range runes {
		t := float64(i)/float64(L-1) + offset
		r, g, b := paletteRGBWrap(sch.colors, t)
		buf.WriteString(ansiRGB(r, g, b))
		buf.WriteRune(ch)
	}
	buf.WriteString(reset)
	return buf.String()
}

// one color code per line so field labels match the logo
func labelColor(sch scheme, lineIdx int) string {
	if len(sch.colors) == 0 {
		return orange
	}
	code := sch.colors[lineIdx%len(sch.colors)]
	if useTrueColor() {
		r, g, b := ansi256ToRGB(code)
		return ansiRGB(r, g, b)
	}
	return ansi(code)
}

// merges ~/.config/hackfetch/colors.json into schemes
// user entries override builtins with the same name
// format
//   { "schemes": { "mytheme": { "colors": [196, 202, 208], "mode": "per-line" } } }
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
		dbg("custom schemes parse error %v", err)
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

// -color auto = pride in june hackclub the rest of the year
func resolveAutoScheme() string {
	if time.Now().Month() == time.June {
		return "pride"
	}
	return "hackclub"
}


// ansi 256-color -> hex string
// shared by the svg exporter and the png/jpg rasterizer so both produce identical colors
func ansi256ToHex(code int) string {
	r, g, b := ansi256ToRGB(code)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// same math as ansi256ToHex but returns the raw channels
// truecolor path + interpolation helpers work in RGB space
func ansi256ToRGB(code int) (uint8, uint8, uint8) {
	if code < 0 || code > 255 {
		return 0xec, 0xec, 0xec
	}
	if code < 16 {
		std := [16][3]uint8{
			{0x00, 0x00, 0x00}, {0xcc, 0x00, 0x00}, {0x4e, 0x9a, 0x06}, {0xc4, 0xa0, 0x00},
			{0x34, 0x65, 0xa4}, {0x75, 0x50, 0x7b}, {0x06, 0x98, 0x9a}, {0xd3, 0xd7, 0xcf},
			{0x55, 0x57, 0x53}, {0xef, 0x29, 0x29}, {0x8a, 0xe2, 0x34}, {0xfc, 0xe9, 0x4f},
			{0x72, 0x9f, 0xcf}, {0xad, 0x7f, 0xa8}, {0x34, 0xe2, 0xe2}, {0xee, 0xee, 0xec},
		}
		c := std[code]
		return c[0], c[1], c[2]
	}
	if code < 232 {
		n := code - 16
		r := n / 36
		g := (n / 6) % 6
		b := n % 6
		v := func(x int) uint8 {
			if x == 0 {
				return 0
			}
			return uint8(55 + x*40)
		}
		return v(r), v(g), v(b)
	}
	v := uint8(8 + (code-232)*10)
	return v, v, v
}

// linear interpolate between two RGB colors
// not perceptually uniform but plenty smooth for terminal gradients
// TODO oklab someday for actually correct color blending
func lerpColor(r1, g1, b1, r2, g2, b2 uint8, t float64) (uint8, uint8, uint8) {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	mix := func(a, b uint8) uint8 {
		return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5)
	}
	return mix(r1, r2), mix(g1, g2), mix(b1, b2)
}

// interpolated RGB at position t in the palette
// treats the palette as cyclic (t=0 and t=1 map to the same color)
// used wherever a gradient needs to loop cleanly
func paletteRGBWrap(palette []int, t float64) (uint8, uint8, uint8) {
	n := len(palette)
	if n == 0 {
		return 0xec, 0xec, 0xec
	}
	if n == 1 {
		return ansi256ToRGB(palette[0])
	}
	// wrap t into [0, 1)
	t = t - math.Floor(t)
	pos := t * float64(n)
	lo := int(pos) % n
	hi := (lo + 1) % n
	frac := pos - math.Floor(pos)
	r1, g1, b1 := ansi256ToRGB(palette[lo])
	r2, g2, b2 := ansi256ToRGB(palette[hi])
	return lerpColor(r1, g1, b1, r2, g2, b2, frac)
}
