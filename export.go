package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed assets/DejaVuSansMono.ttf
var embeddedFontData []byte

// strip ansi escape sequences so text can be embedded in SVG/PNG
// without dumping raw control codes into the output
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

// hex fill for a given logo row
func svgLineColor(sch scheme, lineIdx int) string {
	if len(sch.colors) == 0 {
		return "#ff8c3a"
	}
	return ansi256ToHex(sch.colors[lineIdx%len(sch.colors)])
}

// one logo row as a run of <tspan> elements
// per-char schemes = one tspan per glyph with a smoothly interpolated fill
// so the exported card matches the truecolor terminal render exactly
func svgLogoLine(sch scheme, line string, lineIdx int) string {
	if sch.mode != modePerChar {
		return fmt.Sprintf(`<tspan fill="%s">%s</tspan>`,
			svgLineColor(sch, lineIdx), escapeXML(line))
	}
	runes := []rune(line)
	L := len(runes)
	if L == 0 {
		return ""
	}
	offset := float64(lineIdx) / float64(len(sch.colors)*3)
	var b strings.Builder
	for i, r := range runes {
		var t float64
		if L == 1 {
			t = offset
		} else {
			t = float64(i)/float64(L-1) + offset
		}
		red, gr, bl := paletteRGBWrap(sch.colors, t)
		fmt.Fprintf(&b, `<tspan fill="#%02x%02x%02x">%s</tspan>`, red, gr, bl, escapeXML(string(r)))
	}
	return b.String()
}

// writes the fetch as an svg card
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

// cached font face so we dont re-parse the embedded ttf on every export
var cachedRasterFace font.Face

func rasterFontFace() (font.Face, error) {
	if cachedRasterFace != nil {
		return cachedRasterFace, nil
	}
	parsed, err := opentype.Parse(embeddedFontData)
	if err != nil {
		return nil, fmt.Errorf("parse embedded font %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    16,
		DPI:     144,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create font face %w", err)
	}
	cachedRasterFace = face
	return face, nil
}

func hexToRGBA(hex string) color.RGBA {
	if len(hex) < 7 || hex[0] != '#' {
		return color.RGBA{0xec, 0xec, 0xec, 0xff}
	}
	r, _ := strconv.ParseUint(hex[1:3], 16, 8)
	g, _ := strconv.ParseUint(hex[3:5], 16, 8)
	b, _ := strconv.ParseUint(hex[5:7], 16, 8)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 0xff}
}


// paints a rectangle with rounded corners onto an RGBA image
// only the four corner squares run the distance check the interior fills straight
func fillRoundedRect(img *image.RGBA, r image.Rectangle, radius int, col color.Color) {
	rr := radius * radius
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			var dx, dy int
			switch {
			case x < r.Min.X+radius && y < r.Min.Y+radius:
				dx = (r.Min.X + radius) - x
				dy = (r.Min.Y + radius) - y
			case x >= r.Max.X-radius && y < r.Min.Y+radius:
				dx = x - (r.Max.X - radius - 1)
				dy = (r.Min.Y + radius) - y
			case x < r.Min.X+radius && y >= r.Max.Y-radius:
				dx = (r.Min.X + radius) - x
				dy = y - (r.Max.Y - radius - 1)
			case x >= r.Max.X-radius && y >= r.Max.Y-radius:
				dx = x - (r.Max.X - radius - 1)
				dy = y - (r.Max.Y - radius - 1)
			default:
				img.Set(x, y, col)
				continue
			}
			if dx*dx+dy*dy <= rr {
				img.Set(x, y, col)
			}
		}
	}
}

func drawRasterText(img *image.RGBA, face font.Face, x, y int, s string, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

// one logo row respecting per-line vs per-char color modes
// per-char uses the same smooth palette interpolation as the terminal + svg paths
func drawRasterLogoLine(img *image.RGBA, face font.Face, x, y int, line string, sch scheme, lineIdx, charAdvance int) {
	if sch.mode != modePerChar {
		drawRasterText(img, face, x, y, line, hexToRGBA(svgLineColor(sch, lineIdx)))
		return
	}
	runes := []rune(line)
	L := len(runes)
	if L == 0 {
		return
	}
	offset := float64(lineIdx) / float64(len(sch.colors)*3)
	for i, r := range runes {
		var t float64
		if L == 1 {
			t = offset
		} else {
			t = float64(i)/float64(L-1) + offset
		}
		rr, gg, bb := paletteRGBWrap(sch.colors, t)
		drawRasterText(img, face, x+i*charAdvance, y, string(r), color.RGBA{rr, gg, bb, 0xff})
	}
}

// writes the fetch as png or jpeg
// layout mirrors exportSVG so both formats look the same to the eye
func exportRaster(path string, logoLines []string, sch scheme, fields []field, format string) error {
	face, err := rasterFontFace()
	if err != nil {
		return err
	}

	// derive char advance from the font (monospace guarantees its uniform)
	advM, _ := face.GlyphAdvance('M')
	charW := advM.Ceil()
	if charW < 1 {
		charW = 14
	}
	lineH := face.Metrics().Height.Ceil() + 6
	if lineH < 20 {
		lineH = 32
	}
	padding := 44
	const (
		logoW  = 33
		sepW   = 3
		fieldW = 50
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

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// transparent outside the rounded rect so PNG has clean corners
	// JPEG flattens it below
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)
	fillRoundedRect(img, img.Bounds(), 24, color.RGBA{0x0a, 0x0a, 0x0c, 0xff})

	textColor := color.RGBA{0xec, 0xec, 0xec, 0xff}
	dimColor := color.RGBA{0x75, 0x75, 0x80, 0xff}

	for i := 0; i < rows; i++ {
		y := padding + (i+1)*lineH

		if i < len(logoLines) {
			drawRasterLogoLine(img, face, padding, y, logoLines[i], sch, i, charW)
		}

		if i < len(fields) {
			f := fields[i]
			x := padding + (logoW+sepW)*charW
			labelClean := stripANSI(f.label)
			valueClean := stripANSI(f.value)
			if labelClean == "" {
				drawRasterText(img, face, x, y, valueClean, textColor)
			} else {
				drawRasterText(img, face, x, y, labelClean, hexToRGBA(svgLineColor(sch, i)))
				drawRasterText(img, face, x+13*charW, y, valueClean, textColor)
			}
		}
	}

	footerY := padding + (rows+1)*lineH + 12
	drawRasterText(img, face, padding, footerY, "hackfetch · github.com/xerneas3318/hackfetch", dimColor)

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	switch format {
	case "png":
		return png.Encode(out, img)
	case "jpeg":
		// jpeg has no alpha so flatten the transparent corners onto the card bg first
		flat := image.NewRGBA(img.Bounds())
		draw.Draw(flat, flat.Bounds(), &image.Uniform{color.RGBA{0x0a, 0x0a, 0x0c, 0xff}}, image.Point{}, draw.Src)
		draw.Draw(flat, flat.Bounds(), img, image.Point{}, draw.Over)
		return jpeg.Encode(out, flat, &jpeg.Options{Quality: 92})
	default:
		return fmt.Errorf("unknown raster format %s", format)
	}
}
