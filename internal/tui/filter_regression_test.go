package tui

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func TestKeyboardSearchSelectsMatchAndReturnsToFullList(t *testing.T) {
	model := updateModel(t, NewModel(testAssessments()), keyPress("/"))
	updated, command := model.Update(keyPress("absent"))
	model = updated.(Model)
	matches := searchMatches(t, command)
	model = updateModel(t, model, matches)
	visible := model.list.VisibleItems()
	if len(visible) != 1 || visible[0].FilterValue() != "absent" {
		t.Fatalf("search matches = %v, want only absent", visible)
	}
	model = updateModel(t, model, keyPress("enter"))
	model = updateModel(t, model, keyPress("space"))
	model = updateModel(t, model, keyPress("esc"))
	assertSearchReturnedToFullList(t, model)
	if !model.selected["absent"] || !model.selected["installed"] {
		t.Fatalf("selection after clearing search = %v, want absent and installed", model.selected)
	}
}

func assertSearchReturnedToFullList(t *testing.T, model Model) {
	t.Helper()
	if model.phase != phaseList {
		t.Fatalf("phase after clearing search = %q, want list", model.phase)
	}
	if model.list.FilterState() != list.Unfiltered {
		t.Fatalf("filter state after escape = %v, want unfiltered", model.list.FilterState())
	}
	if visible := model.list.VisibleItems(); len(visible) != len(testAssessments()) {
		t.Fatalf("visible skills after escape = %d, want %d", len(visible), len(testAssessments()))
	}
}

func searchMatches(t *testing.T, command tea.Cmd) list.FilterMatchesMsg {
	t.Helper()
	if command == nil {
		t.Fatal("search command = nil, want asynchronous filtering")
	}
	message := command()
	if matches, ok := message.(list.FilterMatchesMsg); ok {
		return matches
	}
	if batch, ok := message.(tea.BatchMsg); ok {
		return batchSearchMatches(t, batch)
	}
	t.Fatalf("search message = %T, want filter matches", message)
	return nil
}

func batchSearchMatches(t *testing.T, batch tea.BatchMsg) list.FilterMatchesMsg {
	t.Helper()
	for _, child := range batch {
		if matches, ok := child().(list.FilterMatchesMsg); ok {
			return matches
		}
	}
	t.Fatal("command batch did not contain filter matches")
	return nil
}
