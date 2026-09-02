package ui

import "charm.land/lipgloss/v2"

const (
	cgaRed         lipgloss.ANSIColor = 1
	cgaGray        lipgloss.ANSIColor = 8
	cgaBrightGreen lipgloss.ANSIColor = 10
	cgaBrightCyan  lipgloss.ANSIColor = 14

	terminalShopSelectionForeground lipgloss.ANSIColor = 102
	terminalShopSelectionBackground lipgloss.ANSIColor = 202
)

var (
	selectionStyle = lipgloss.NewStyle().
			Foreground(terminalShopSelectionForeground).
			Background(terminalShopSelectionBackground)
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
