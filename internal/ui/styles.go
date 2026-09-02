package ui

import "charm.land/lipgloss/v2"

const (
	cgaBlack        lipgloss.ANSIColor = 0
	cgaRed          lipgloss.ANSIColor = 1
	cgaGray         lipgloss.ANSIColor = 8
	cgaBrightGreen  lipgloss.ANSIColor = 10
	cgaBrightYellow lipgloss.ANSIColor = 11
	cgaBrightCyan   lipgloss.ANSIColor = 14

	terminalShopOrange lipgloss.ANSIColor = 202
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true)
	promptStyle = lipgloss.NewStyle().
			Foreground(cgaBrightGreen).
			Bold(true)
	primaryCopyStyle   = lipgloss.NewStyle()
	secondaryCopyStyle = lipgloss.NewStyle().
				Foreground(cgaGray)
	successStyle = lipgloss.NewStyle().
			Foreground(cgaBrightGreen)
	errorStyle = lipgloss.NewStyle().
			Foreground(cgaRed).
			Bold(true)
	terminalStateStyle = lipgloss.NewStyle().
				Foreground(cgaBrightCyan)
)

// sectionAccent gives each section its own colour so the inverted selection bar
// and the active header cell agree about which section the visitor is in.
func sectionAccent(section Section) lipgloss.ANSIColor {
	switch section {
	case SectionProjects:
		return terminalShopOrange
	case SectionResearch:
		return cgaBrightCyan
	case SectionContact:
		return cgaBrightGreen
	default:
		return cgaBrightYellow
	}
}

// selectionStyle inverts the section accent: the accent becomes the background
// and the text drops to black. This is the only background-painted element in
// the interface, so it is unambiguously where the visitor's cursor is.
func selectionStyle(accent lipgloss.ANSIColor) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(cgaBlack).
		Background(accent)
}

// activeHeaderStyle marks the current section in the header with the same
// accent, but foreground-only, so it never competes with the selection bar.
func activeHeaderStyle(accent lipgloss.ANSIColor) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)
}
