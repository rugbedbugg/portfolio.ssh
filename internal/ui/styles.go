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

// selectionStyle inverts the accent: orange becomes the background and the text
// drops to black. This is the only background-painted element in the interface,
// so it is unambiguously where the visitor's cursor is.
var selectionStyle = lipgloss.NewStyle().
	Foreground(cgaBlack).
	Background(terminalShopOrange)

// activeHeaderStyle marks the current section in the header with the same
// accent, but foreground-only, so it never competes with the selection bar.
var activeHeaderStyle = lipgloss.NewStyle().
	Foreground(terminalShopOrange).
	Bold(true)

// bannerStyle paints the name banner in the accent, foreground-only.
var bannerStyle = lipgloss.NewStyle().
	Foreground(terminalShopOrange).
	Bold(true)
