package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// bannerRows is the height of the block face, and so of the banner it draws.
const bannerRows = 5

// moneyRows is the height of the money face. It is the taller of the two, so
// renderNameplate only reaches for it when the extra row is free.
const moneyRows = 6

// bannerFace is one drawable typeface: a fixed row count and a glyph per
// supported rune. Glyphs are rectangular — every row of a glyph is the same
// width — but different glyphs may have different widths, which is what lets a
// full stop stay narrow next to a full-width capital.
type bannerFace struct {
	rows   int
	glyphs map[rune][]string
	// separator is written between two adjacent glyphs. Faces that build their
	// own side bearing into each glyph return an empty string.
	separator func(left, right rune) string
}

// moneyFace is a six-row face in the style of the FIGlet "money" font: dollar
// stems with slashed shoulders and an underscored foot. Glyphs carry their own
// side bearing, so nothing is inserted between them.
//
// It covers only the runes in the profile name. renderBanner returns an empty
// string for anything else, which drops renderNameplate to the block face — so
// an unlisted rune degrades rather than breaking.
var moneyFace = bannerFace{
	rows:      moneyRows,
	separator: func(rune, rune) string { return "" },
	glyphs: map[rune][]string{
		'P': {
			" /$$$$$ ",
			"| $$__$$",
			"| $$$$$/",
			"| $$__/ ",
			"| $$    ",
			"|__/    ",
		},
		'A': {
			"  /$$$$  ",
			" /$$__$$ ",
			"| $$  $$ ",
			"| $$$$$$ ",
			"| $$__$$ ",
			"|__/ |__/",
		},
		'R': {
			" /$$$$$$  ",
			"| $$__  $$",
			"| $$$$$$$/",
			"| $$__  $$",
			"| $$  \\ $$",
			"|__/  \\__/",
		},
		'T': {
			" /$$$$$$",
			"|_  $$_/",
			"  | $$  ",
			"  | $$  ",
			"  | $$  ",
			"  |__/  ",
		},
		'H': {
			" /$$   /$$",
			"| $$  | $$",
			"| $$$$$$$$",
			"| $$__  $$",
			"| $$  | $$",
			"|__/  |__/",
		},
		'G': {
			"  /$$$$$$ ",
			" /$$__  $$",
			"| $$  \\__/",
			"| $$ /$$$$",
			"| $$|_  $$",
			" \\______/ ",
		},
		'.': {
			"    ",
			"    ",
			"    ",
			"    ",
			" /$$",
			"|__/",
		},
		' ': {
			"   ",
			"   ",
			"   ",
			"   ",
			"   ",
			"   ",
		},
	},
}

// blockFace is the compact five-row fallback drawn with U+2588 FULL BLOCK. It
// covers the whole alphabet and is narrow enough for terminals that cannot
// afford the money face.
var blockFace = bannerFace{
	rows:      bannerRows,
	separator: blockSeparator,
	glyphs: map[rune][]string{
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
		// Two columns wide with the dot on the right: the leading blank keeps
		// the dot from merging into the bottom stroke of a preceding glyph
		// such as G.
		'.': {"  ", "  ", "  ", "  ", " █"},
		// Three columns centered: a stroke spanning the full cell would read
		// as a rule joining its neighbours rather than as a hyphen.
		'-': {"     ", "     ", " ███ ", "     ", "     "},
		' ': {"   ", "   ", "   ", "   ", "   "},
	},
}

// blockSeparator spaces block glyphs apart, except that a full stop hugs the
// glyph it follows and the one it precedes, so "P.G." reads as one
// abbreviation rather than as four spaced-out characters.
func blockSeparator(left, right rune) string {
	if left == '.' || right == '.' {
		return ""
	}
	return " "
}

// renderBanner draws name in face, unstyled — renderNameplate owns the styling
// for every tier. It returns an empty string when a character has no glyph or
// the result would not fit in width, so the caller falls back to a smaller
// tier rather than rendering a broken banner.
func renderBanner(face bannerFace, name string, width int) string {
	characters := []rune(strings.ToUpper(name))
	glyphs := make([][]string, 0, len(characters))
	for _, character := range characters {
		glyph, ok := face.glyphs[character]
		if !ok {
			return ""
		}
		glyphs = append(glyphs, glyph)
	}
	if len(glyphs) == 0 {
		return ""
	}

	rows := make([]string, face.rows)
	for row := range rows {
		line := strings.Builder{}
		for index, glyph := range glyphs {
			if index > 0 {
				line.WriteString(face.separator(characters[index-1], characters[index]))
			}
			line.WriteString(glyph[row])
		}
		rows[row] = line.String()
	}
	if lipgloss.Width(rows[0]) > width {
		return ""
	}
	return strings.Join(rows, "\n")
}
