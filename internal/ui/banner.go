package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// bannerRows is the height of one banner glyph, and so of the whole banner.
const bannerRows = 5

// bannerGlyphGap separates adjacent glyphs. A full stop hugs the glyph it
// follows and the one it precedes, so "P.G." reads as one abbreviation rather
// than as four spaced-out characters.
const bannerGlyphGap = " "

func bannerSeparator(left, right rune) string {
	if left == '.' || right == '.' {
		return ""
	}
	return bannerGlyphGap
}

// bannerFont is a five-row block font drawn with U+2588 FULL BLOCK. Glyphs are
// five columns wide apart from the deliberately narrow space and full stop, so
// "P.G." reads as an abbreviation rather than as separated letters.
var bannerFont = map[rune][bannerRows]string{
	'A': {" ███ ", "█   █", "█████", "█   █", "█   █"},
	'B': {"████ ", "█   █", "████ ", "█   █", "████ "},
	'C': {" ████", "█    ", "█    ", "█    ", " ████"},
	'D': {"████ ", "█   █", "█   █", "█   █", "████ "},
	'E': {"█████", "█    ", "████ ", "█    ", "█████"},
	'F': {"█████", "█    ", "████ ", "█    ", "█    "},
	'G': {" ████", "█    ", "█  ██", "█   █", " ████"},
	'H': {"█   █", "█   █", "█████", "█   █", "█   █"},
	'I': {"█████", "  █  ", "  █  ", "  █  ", "█████"},
	'J': {"    █", "    █", "    █", "█   █", " ███ "},
	'K': {"█   █", "█  █ ", "███  ", "█  █ ", "█   █"},
	'L': {"█    ", "█    ", "█    ", "█    ", "█████"},
	'M': {"█   █", "██ ██", "█ █ █", "█   █", "█   █"},
	'N': {"█   █", "██  █", "█ █ █", "█  ██", "█   █"},
	'O': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'P': {"████ ", "█   █", "████ ", "█    ", "█    "},
	'Q': {" ███ ", "█   █", "█   █", "█  █ ", " ██ █"},
	'R': {"████ ", "█   █", "████ ", "█  █ ", "█   █"},
	'S': {" ████", "█    ", " ███ ", "    █", "████ "},
	'T': {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
	'U': {"█   █", "█   █", "█   █", "█   █", " ███ "},
	'V': {"█   █", "█   █", "█   █", " █ █ ", "  █  "},
	'W': {"█   █", "█   █", "█ █ █", "██ ██", "█   █"},
	'X': {"█   █", " █ █ ", "  █  ", " █ █ ", "█   █"},
	'Y': {"█   █", " █ █ ", "  █  ", "  █  ", "  █  "},
	'Z': {"█████", "   █ ", "  █  ", " █   ", "█████"},
	// Two columns wide with the dot on the right: the leading blank keeps the
	// dot from merging into the bottom stroke of a preceding glyph such as G.
	'.': {"  ", "  ", "  ", "  ", " █"},
	'-': {"     ", "     ", "█████", "     ", "     "},
	' ': {"   ", "   ", "   ", "   ", "   "},
}

// renderBanner draws name in the block font. It returns an empty string when a
// character has no glyph or the result would not fit in width, so the caller
// simply omits the banner rather than rendering a broken one.
func renderBanner(name string, width int) string {
	characters := []rune(strings.ToUpper(name))
	glyphs := make([][bannerRows]string, 0, len(characters))
	for _, character := range characters {
		glyph, ok := bannerFont[character]
		if !ok {
			return ""
		}
		glyphs = append(glyphs, glyph)
	}
	if len(glyphs) == 0 {
		return ""
	}

	rows := make([]string, bannerRows)
	for row := range rows {
		line := strings.Builder{}
		for index, glyph := range glyphs {
			if index > 0 {
				line.WriteString(bannerSeparator(characters[index-1], characters[index]))
			}
			line.WriteString(glyph[row])
		}
		rows[row] = line.String()
	}
	if lipgloss.Width(rows[0]) > width {
		return ""
	}
	return bannerStyle.Render(strings.Join(rows, "\n"))
}
