package tui

import "charm.land/lipgloss/v2"

type viewStyles struct {
	title    lipgloss.Style
	subtitle lipgloss.Style
	muted    lipgloss.Style
	focused  lipgloss.Style
	selected lipgloss.Style
	install  lipgloss.Style
	remove   lipgloss.Style
	warning  lipgloss.Style
}

func newViewStyles() viewStyles {
	return viewStyles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan),
		subtitle: lipgloss.NewStyle().Faint(true),
		muted:    lipgloss.NewStyle().Faint(true),
		focused:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan),
		selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Magenta),
		install:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green),
		remove:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red),
		warning:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow),
	}
}
