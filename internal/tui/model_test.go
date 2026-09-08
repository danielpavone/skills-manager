package tui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/project"
)

func TestModelStartsWithInstalledSkillsSelectedAndConflictsLocked(t *testing.T) {
	model := NewModel(testAssessments())
	if !model.selected["installed"] || model.selected["absent"] {
		t.Fatalf("initial selection = %#v, want only installed", model.selected)
	}
	view := model.View().Content
	assertContains(t, view, "installed / keep")
	assertContains(t, view, "wrong_target / locked")
	assertContains(t, view, "[!]")
}

func TestModelFiltersCatalogByNameAndRetainsKeyboardPaging(t *testing.T) {
	model := NewModel([]project.LinkAssessment{
		{Skill: catalog.Skill{Name: "alpha", SourcePath: "/catalog/alpha"}, LinkPath: "/project/.agents/skills/alpha", State: project.LinkAbsent},
		{Skill: catalog.Skill{Name: "beta", SourcePath: "/catalog/beta"}, LinkPath: "/project/.agents/skills/beta", State: project.LinkAbsent},
		{Skill: catalog.Skill{Name: "gamma", SourcePath: "/catalog/gamma"}, LinkPath: "/project/.agents/skills/gamma", State: project.LinkAbsent},
	})
	model.list.SetFilterText("beta")
	if visible := model.list.VisibleItems(); len(visible) != 1 {
		t.Fatalf("filtered items = %d, want 1", len(visible))
	}
	model.list.ResetFilter()
	model = updateModel(t, model, keyPress("down"))
	item, ok := model.list.SelectedItem().(skillItem)
	if !ok || item.assessment.Skill.Name != "beta" {
		t.Fatalf("selected item = %#v, want beta", model.list.SelectedItem())
	}
}

func TestModelUpdatesSelectionConfirmationAndCancellationByKeyboard(t *testing.T) {
	model := NewModel(testAssessments())
	model = updateModel(t, model, keyPress("space"))
	if model.selected["installed"] {
		t.Fatal("space did not toggle the focused installed skill")
	}
	model = updateModel(t, model, keyPress("down"))
	model = updateModel(t, model, keyPress("space"))
	model = updateModel(t, model, keyPress("enter"))
	if model.phase != phaseConfirm {
		t.Fatalf("phase after enter = %q, want confirm", model.phase)
	}
	model = updateModel(t, model, keyPress("esc"))
	if model.phase != phaseList {
		t.Fatalf("phase after escape = %q, want list", model.phase)
	}
	model = updateModel(t, model, keyPress("q"))
	if model.phase != phaseCanceled {
		t.Fatalf("phase after q = %q, want canceled", model.phase)
	}
}

func TestModelResizeKeepsListWithinSmallTerminal(t *testing.T) {
	model := NewModel(testAssessments())
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 24, Height: 5})
	if model.width != 24 || model.height != 5 || model.list.Width() != 24 {
		t.Fatalf("size = %dx%d, list width = %d", model.width, model.height, model.list.Width())
	}
	if strings.TrimSpace(model.View().Content) == "" {
		t.Fatal("small terminal rendered an empty view")
	}
}

func TestModelSelectionExcludesLockedStates(t *testing.T) {
	model := NewModel(testAssessments())
	selection, err := model.Selection()
	if err != nil {
		t.Fatalf("Selection() error = %v", err)
	}
	if len(selection.SelectedNames) != 1 || selection.SelectedNames[0] != "installed" {
		t.Fatalf("selection = %#v, want installed only", selection.SelectedNames)
	}
}

func TestModelRendersSummaryState(t *testing.T) {
	model := NewModel(nil).SetSummary(project.BatchResult{Results: []project.OperationResult{
		{SkillName: "skill", Outcome: project.OutcomeInstalled},
	}})
	assertContains(t, model.View().Content, "instaladas:\n- skill")
}

func TestSelectionUICompletesControlledKeyboardSession(t *testing.T) {
	output := &bytes.Buffer{}
	ui := NewSelectionUI(strings.NewReader("\ry"), output)
	selection, err := ui.Choose(context.Background(), []project.LinkAssessment{
		{Skill: catalog.Skill{Name: "absent", SourcePath: "/catalog/absent"}, LinkPath: "/project/.agents/skills/absent", State: project.LinkAbsent},
	})
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if len(selection.SelectedNames) != 0 {
		t.Fatalf("selection = %#v, want no selected skills", selection.SelectedNames)
	}
}

func TestSelectionUICompletesTenSkillKeyboardSession(t *testing.T) {
	assessments := make([]project.LinkAssessment, 10)
	for index := range assessments {
		name := "skill-" + string(rune('a'+index))
		assessments[index] = project.LinkAssessment{
			Skill:    catalog.Skill{Name: name, SourcePath: "/catalog/" + name},
			LinkPath: "/project/.agents/skills/" + name,
			State:    project.LinkAbsent,
		}
	}
	input := " " + strings.Repeat("\x1b[B ", len(assessments)-1) + "\ry"
	selection, err := NewSelectionUI(strings.NewReader(input), &bytes.Buffer{}).Choose(context.Background(), assessments)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if len(selection.SelectedNames) != len(assessments) {
		t.Fatalf("selected skills = %d, want %d", len(selection.SelectedNames), len(assessments))
	}
}

func updateModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(msg)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model = %T, want tui.Model", updated)
	}
	return result
}

func keyPress(name string) tea.KeyPressMsg {
	if name == "space" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeySpace})
	}
	if name == "enter" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})
	}
	return tea.KeyPressMsg(tea.Key{Text: name})
}

func testAssessments() []project.LinkAssessment {
	return []project.LinkAssessment{
		{Skill: catalog.Skill{Name: "installed", SourcePath: "/catalog/installed"}, LinkPath: "/project/.agents/skills/installed", State: project.LinkInstalled},
		{Skill: catalog.Skill{Name: "absent", SourcePath: "/catalog/absent"}, LinkPath: "/project/.agents/skills/absent", State: project.LinkAbsent},
		{Skill: catalog.Skill{Name: "wrong", SourcePath: "/catalog/wrong"}, LinkPath: "/project/.agents/skills/wrong", State: project.LinkWrongTarget},
	}
}

func assertContains(t *testing.T, text, expected string) {
	t.Helper()
	if !strings.Contains(text, expected) {
		t.Fatalf("view %q does not contain %q", text, expected)
	}
}
