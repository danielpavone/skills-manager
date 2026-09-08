package tui

import tea "charm.land/bubbletea/v2"

func isKey(msg tea.KeyPressMsg, names ...string) bool {
	for _, name := range names {
		if msg.String() == name {
			return true
		}
	}
	return false
}

func isCancelKey(msg tea.KeyPressMsg) bool {
	return isKey(msg, "q", "ctrl+c", "esc")
}

func isConfirmKey(msg tea.KeyPressMsg) bool {
	return isKey(msg, "y", "enter")
}
