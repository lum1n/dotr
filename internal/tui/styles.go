package tui

import (
	"charm.land/lipgloss/v2"
)

type styles struct {
	title       lipgloss.Style
	status      lipgloss.Style
	panel       lipgloss.Style
	panelActive lipgloss.Style
	listCursor  lipgloss.Style
	listNormal  lipgloss.Style
	listApp     lipgloss.Style
	muted       lipgloss.Style
	parseOK     lipgloss.Style
	parseFail   lipgloss.Style
	secret      lipgloss.Style
}

func newStyles() styles {
	accent := lipgloss.Color("39")
	muted := lipgloss.Color("245")
	border := lipgloss.Color("238")
	activeBorder := lipgloss.Color("39")

	return styles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1),
		status: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),
		panelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeBorder).
			Padding(0, 1),
		listCursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Bold(true),
		listNormal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		listApp: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),
		muted: lipgloss.NewStyle().Foreground(muted),
		parseOK: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true),
		parseFail: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		secret: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
	}
}
