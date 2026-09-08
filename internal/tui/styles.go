package tui

import "charm.land/lipgloss/v2"

type viewStyles struct {
	title   lipgloss.Style
	muted   lipgloss.Style
	warning lipgloss.Style
}

func newViewStyles() viewStyles {
	return viewStyles{
		title:   lipgloss.NewStyle().Bold(true),
		muted:   lipgloss.NewStyle().Faint(true),
		warning: lipgloss.NewStyle().Bold(true),
	}
}
