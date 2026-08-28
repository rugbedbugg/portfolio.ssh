package ui

import "charm.land/lipgloss/v2"

const (
	cgaBlack       lipgloss.ANSIColor = 0
	cgaGreen       lipgloss.ANSIColor = 2
	cgaCyan        lipgloss.ANSIColor = 3
	cgaRed         lipgloss.ANSIColor = 4
	cgaGray        lipgloss.ANSIColor = 7
	cgaBrightGreen lipgloss.ANSIColor = 10
	cgaBrightCyan  lipgloss.ANSIColor = 11
	cgaWhite       lipgloss.ANSIColor = 15
)

var (
	borderStyle = lipgloss.NewStyle().
			Background(cgaBlack).
			Border(lipgloss.NormalBorder()).
			BorderForeground(cgaCyan).
			Padding(0, 1)
	selectionStyle = lipgloss.NewStyle().
			Foreground(cgaBrightCyan).
			Background(cgaBlack).
			Bold(true)
	titleStyle = lipgloss.NewStyle().
			Foreground(cgaBrightCyan).
			Background(cgaBlack).
			Bold(true)
	promptStyle = lipgloss.NewStyle().
			Foreground(cgaBrightGreen).
			Background(cgaBlack).
			Bold(true)
	primaryCopyStyle = lipgloss.NewStyle().
				Foreground(cgaWhite).
				Background(cgaBlack)
	secondaryCopyStyle = lipgloss.NewStyle().
				Foreground(cgaGray).
				Background(cgaBlack)
	successStyle = lipgloss.NewStyle().
			Foreground(cgaBrightGreen).
			Background(cgaBlack)
	errorStyle = lipgloss.NewStyle().
			Foreground(cgaRed).
			Background(cgaBlack).
			Bold(true)
	terminalStateStyle = lipgloss.NewStyle().
				Foreground(cgaGreen).
				Background(cgaBlack)
)
