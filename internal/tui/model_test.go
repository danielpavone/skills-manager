package tui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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

func TestNewSkillsListConfiguresSearchPrompt(t *testing.T) {
	configured := newSkillsList(nil, map[string]bool{}, newViewStyles())
	if configured.FilterInput.Prompt != "pesquisa: " {
		t.Fatalf("filter prompt = %q, want pesquisa prompt", configured.FilterInput.Prompt)
	}
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

func TestModelUsesComfortableSpacingAndCompactFallback(t *testing.T) {
	model := NewModel(testAssessments())
	view := ansi.Strip(model.View().Content)
	assertContains(t, view, "skills-manager")
	assertContains(t, view, "Selecione as skills deste projeto\n\n")
	assertContains(t, view, "pressione / para pesquisar\n\n")

	model = updateModel(t, model, tea.WindowSizeMsg{Width: 80, Height: 5})
	if strings.Contains(model.View().Content, "Selecione as skills deste projeto") {
		t.Fatal("compact view includes subtitle, want space reserved for the list")
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

func TestModelConfirmationListsChangesAndAcceptsEnter(t *testing.T) {
	model := NewModel(testAssessments())
	model = updateModel(t, model, keyPress("space"))
	model = updateModel(t, model, keyPress("down"))
	model = updateModel(t, model, keyPress("space"))
	model = updateModel(t, model, keyPress("enter"))
	view := ansi.Strip(model.View().Content)
	assertContains(t, view, "Instalar (1)\n    + absent")
	assertContains(t, view, "Remover (1)\n    - installed")
	assertContains(t, view, "+ absent\n\n  Remover")
	assertContains(t, view, "enter confirmar")

	model = updateModel(t, model, keyPress("enter"))
	if model.phase != phaseSummary {
		t.Fatalf("phase after confirmation enter = %q, want summary", model.phase)
	}
}

func TestModelConfirmationShowsEmptyOperationGroups(t *testing.T) {
	model := NewModel(testAssessments())
	model = updateModel(t, model, keyPress("enter"))
	view := ansi.Strip(model.View().Content)
	assertContains(t, view, "Instalar (0)\n    nenhuma alteração")
	assertContains(t, view, "Remover (0)\n    nenhuma alteração")
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

func TestModelKeepsEveryRenderedLineWithinSmallTerminal(t *testing.T) {
	model := NewModel(testAssessments())
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 24, Height: 5})
	assertLinesFitWidth(t, model.View().Content, 24)
	model = updateModel(t, model, keyPress("enter"))
	assertLinesFitWidth(t, model.View().Content, 24)
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
		{SkillName: "kept", Outcome: project.OutcomeUnchanged},
	}})
	view := ansi.Strip(model.View().Content)
	assertContains(t, view, "operação concluída com sucesso")
	assertContains(t, view, "As skills deste projeto estão atualizadas\n\n")
	assertContains(t, view, "Instaladas (1)\n    + skill")
	assertContains(t, view, "+ skill\n\n  enter voltar às skills")
	assertContains(t, view, "enter voltar às skills • q/esc sair")
	if strings.Contains(view, "kept") || strings.Contains(view, "inalteradas") {
		t.Fatalf("summary view = %q, want unchanged skills omitted", view)
	}
}

func TestModelRendersFailuresAndPreservedConflictsWithGuidance(t *testing.T) {
	model := NewModel(nil).SetSummary(project.BatchResult{HasFailures: true, Results: []project.OperationResult{
		{SkillName: "conflict", Outcome: project.OutcomeUnchanged, ErrorCode: project.ErrorLinkConflict, Message: "destino preservado"},
		{SkillName: "failed", Outcome: project.OutcomeFailed, Message: "permissão negada"},
	}})
	view := ansi.Strip(model.View().Content)
	assertContains(t, view, "operação concluída com falhas")
	assertContains(t, view, "Revise as skills que não puderam ser atualizadas")
	assertContains(t, view, "Conflitos preservados (1)\n    ! conflict: destino preservado")
	assertContains(t, view, "Falhas (1)\n    ! failed: permissão negada")
}

func TestModelSummaryReturnsToUpdatedSkillList(t *testing.T) {
	model := NewModel([]project.LinkAssessment{
		{Skill: catalog.Skill{Name: "skill", SourcePath: "/catalog/skill"}, LinkPath: "/project/.agents/skills/skill", State: project.LinkAbsent},
	})
	model.selected["skill"] = true
	model.list.SetFilterText("skill")
	updated, _ := model.finishApplying(applyResultMsg{result: project.BatchResult{Results: []project.OperationResult{
		{SkillName: "skill", Action: project.ActionInstall, Outcome: project.OutcomeInstalled},
	}}})
	model = updated.(Model)
	model = updateModel(t, model, keyPress("enter"))
	if model.phase != phaseList {
		t.Fatalf("phase after summary enter = %q, want list", model.phase)
	}
	assertContains(t, ansi.Strip(model.View().Content), "installed / keep")
	if model.list.FilterState() != list.Unfiltered {
		t.Fatalf("filter state = %v, want reset list", model.list.FilterState())
	}
}

func TestSelectionUICompletesControlledKeyboardSession(t *testing.T) {
	output := &bytes.Buffer{}
	ui := NewSelectionUI(strings.NewReader("\r\r"), output)
	selection, err := ui.Choose(context.Background(), []project.LinkAssessment{
		{Skill: catalog.Skill{Name: "absent", SourcePath: filepath.Join(os.TempDir(), "catalog", "absent")}, LinkPath: filepath.Join(os.TempDir(), "project", ".agents", "skills", "absent"), State: project.LinkAbsent},
	})
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if len(selection.SelectedNames) != 0 {
		t.Fatalf("selection = %#v, want no selected skills", selection.SelectedNames)
	}
}

func TestModelKeepsAppliedSuccessSummaryOpenUntilQuit(t *testing.T) {
	model := NewModel([]project.LinkAssessment{
		{Skill: catalog.Skill{Name: "installed", SourcePath: "/catalog/installed"}, LinkPath: "/project/installed", State: project.LinkAbsent},
	})
	model.runContext = context.Background()
	model.apply = func(context.Context, project.Selection) (project.BatchResult, error) {
		return project.BatchResult{Results: []project.OperationResult{{SkillName: "installed", Outcome: project.OutcomeInstalled}}}, nil
	}
	model = updateModel(t, model, keyPress("enter"))
	updated, command := model.Update(keyPress("enter"))
	if command == nil {
		t.Fatal("confirmation command is nil, want asynchronous application")
	}
	model = updateModel(t, updated.(Model), command())
	if model.phase != phaseSummary {
		t.Fatalf("phase after application = %q, want summary", model.phase)
	}
	assertContains(t, ansi.Strip(model.View().Content), "operação concluída com sucesso")
	_, quit := model.Update(keyPress("q"))
	if quit == nil {
		t.Fatal("summary quit command is nil, want explicit quit")
	}
}

func TestSelectionUICompletesTenSkillKeyboardSession(t *testing.T) {
	assessments := make([]project.LinkAssessment, 10)
	for index := range assessments {
		name := "skill-" + string(rune('a'+index))
		assessments[index] = project.LinkAssessment{
			Skill:    catalog.Skill{Name: name, SourcePath: filepath.Join(os.TempDir(), "catalog", name)},
			LinkPath: filepath.Join(os.TempDir(), "project", ".agents", "skills", name),
			State:    project.LinkAbsent,
		}
	}
	input := " " + strings.Repeat("\x1b[B ", len(assessments)-1) + "\r\r"
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
	catalogRoot := filepath.Join(os.TempDir(), "catalog")
	projectSkills := filepath.Join(os.TempDir(), "project", ".agents", "skills")
	return []project.LinkAssessment{
		{Skill: catalog.Skill{Name: "installed", SourcePath: filepath.Join(catalogRoot, "installed")}, LinkPath: filepath.Join(projectSkills, "installed"), State: project.LinkInstalled},
		{Skill: catalog.Skill{Name: "absent", SourcePath: filepath.Join(catalogRoot, "absent")}, LinkPath: filepath.Join(projectSkills, "absent"), State: project.LinkAbsent},
		{Skill: catalog.Skill{Name: "wrong", SourcePath: filepath.Join(catalogRoot, "wrong")}, LinkPath: filepath.Join(projectSkills, "wrong"), State: project.LinkWrongTarget},
	}
}

func assertContains(t *testing.T, text, expected string) {
	t.Helper()
	if !strings.Contains(text, expected) {
		t.Fatalf("view %q does not contain %q", text, expected)
	}
}

func assertLinesFitWidth(t *testing.T, view string, expectedWidth int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > expectedWidth {
			t.Fatalf("rendered line width = %d for %q, want at most %d", width, line, expectedWidth)
		}
	}
}
