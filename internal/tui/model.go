package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/danielpavone/skills-manager/internal/project"
)

type phase string

const (
	phaseList     phase = "list"
	phaseConfirm  phase = "confirm"
	phaseApplying phase = "applying"
	phaseSummary  phase = "summary"
	phaseError    phase = "error"
	phaseCanceled phase = "canceled"
)

type Model struct {
	assessments    []project.LinkAssessment
	selected       map[string]bool
	list           list.Model
	styles         viewStyles
	phase          phase
	width          int
	height         int
	result         project.BatchResult
	resultErr      error
	runContext     context.Context
	apply          func(context.Context, project.Selection) (project.BatchResult, error)
	quitAfterApply bool
	hasApplied     bool
}

func NewModel(assessments []project.LinkAssessment) Model {
	selected := initialSelection(assessments)
	items := assessmentItems(assessments)
	styles := newViewStyles()
	return Model{
		assessments: append([]project.LinkAssessment(nil), assessments...),
		selected:    selected,
		list:        newSkillsList(items, selected, styles),
		styles:      styles,
		phase:       phaseList,
		width:       80,
		height:      20,
	}
}

func newSkillsList(items []list.Item, selected map[string]bool, styles viewStyles) list.Model {
	delegate := newSkillDelegate(selected, styles)
	itemsList := list.New(items, delegate, 80, 12)
	itemsList.Title = "skills"
	itemsList.FilterInput.Prompt = "pesquisa: "
	itemsList.SetShowTitle(false)
	itemsList.SetShowStatusBar(false)
	itemsList.SetShowHelp(false)
	itemsList.SetShowFilter(false)
	itemsList.DisableQuitKeybindings()
	return itemsList
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(applyResultMsg); ok {
		return m.finishApplying(result)
	}
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.resize(size.Width, size.Height)
		return m, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m.updateListMessage(msg)
	}
	return m.updateKey(key)
}

func (m Model) updateKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.phase == phaseList {
		return m.updateList(key)
	}
	if m.phase == phaseConfirm {
		return m.updateConfirmation(key)
	}
	if m.phase == phaseApplying && isCancelKey(key) {
		m.quitAfterApply = true
		return m, nil
	}
	if m.phase == phaseSummary && isKey(key, "enter") {
		m.phase = phaseList
		return m, nil
	}
	if (m.phase == phaseSummary || m.phase == phaseError) && isCancelKey(key) {
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateListMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.phase != phaseList {
		return m, nil
	}
	updated, command := m.list.Update(msg)
	m.list = updated
	return m, command
}

func (m Model) updateList(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(key, "ctrl+c"):
		m.phase = phaseCanceled
		return m, tea.Quit
	case m.list.FilterState() == list.FilterApplied && isKey(key, "esc"):
		return m.updateListMessage(key)
	case m.list.FilterState() != list.Filtering && isKey(key, "space"):
		m.toggleCurrent()
		return m, nil
	case m.list.FilterState() != list.Filtering && isKey(key, "enter"):
		m.phase = phaseConfirm
		return m, nil
	case m.list.FilterState() != list.Filtering && isCancelKey(key):
		m.phase = phaseCanceled
		return m, tea.Quit
	default:
		return m.updateListMessage(key)
	}
}

func (m Model) updateConfirmation(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if isConfirmKey(key) {
		if m.apply != nil {
			selection, err := m.Selection()
			if err != nil {
				m.resultErr = err
				m.phase = phaseError
				return m, nil
			}
			m.phase = phaseApplying
			return m, m.applySelection(selection)
		}
		m.phase = phaseSummary
		return m, tea.Quit
	}
	if isCancelKey(key) {
		if key.String() == "esc" {
			m.phase = phaseList
			return m, nil
		}
		m.phase = phaseCanceled
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) resize(width, height int) {
	m.width = maxWidth(width)
	m.height = maxWidth(height)
	reservedRows := 9
	if m.height < 12 {
		reservedRows = 4
	}
	m.list.SetSize(m.width, maxWidth(m.height-reservedRows))
}

func (m *Model) toggleCurrent() {
	item, ok := m.list.SelectedItem().(skillItem)
	if !ok || !isMutable(item.assessment.State) {
		return
	}
	name := item.assessment.Skill.Name
	m.selected[name] = !m.selected[name]
}

func (m Model) View() tea.View {
	content := m.renderContent()
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m Model) renderContent() string {
	switch m.phase {
	case phaseConfirm:
		return m.renderConfirmation()
	case phaseApplying:
		return fitViewWidth([]string{"", "  " + m.styles.title.Render("aplicando alterações..."), ""}, m.width)
	case phaseSummary:
		return m.renderResult()
	case phaseError:
		return fitViewWidth([]string{"", "  " + m.styles.title.Render("falha ao aplicar alterações"), "", "  " + m.resultErr.Error(), "", "  " + m.styles.muted.Render("q/esc sair")}, m.width)
	case phaseCanceled:
		return "operação cancelada"
	default:
		return m.renderList()
	}
}

func (m Model) renderList() string {
	title := "  " + m.styles.title.Render("skills-manager")
	search := "  " + m.styles.muted.Render("pressione / para pesquisar")
	if m.list.FilterState() != list.Unfiltered {
		search = "  " + m.list.FilterInput.View()
	}
	help := "  " + m.styles.muted.Render("↑/↓ ou j/k mover • space selecionar • enter revisar • esc/q sair")
	if m.height < 12 {
		return fitViewWidth([]string{title, search, m.list.View(), help}, m.width)
	}
	subtitle := "  " + m.styles.subtitle.Render("Selecione as skills deste projeto")
	return fitViewWidth([]string{"", title, subtitle, "", search, "", m.list.View(), "", help}, m.width)
}

func (m Model) renderConfirmation() string {
	installNames, removeNames, err := confirmationSkillNames(m.assessments, m.selected)
	if err != nil {
		return fitViewWidth([]string{m.styles.title.Render("confirmar alterações"), err.Error()}, m.width)
	}
	installBlock := formatConfirmationGroup("Instalar", "+", installNames, m.styles.install)
	removeBlock := formatConfirmationGroup("Remover", "-", removeNames, m.styles.remove)
	locked := "  " + m.styles.warning.Render(fmt.Sprintf("Conflitos bloqueados (%d)", countLocked(m.assessments)))
	lines := []string{"", "  " + m.styles.title.Render("confirmar alterações"), "  " + m.styles.subtitle.Render("Revise o que será alterado antes de continuar"), "", installBlock, "", removeBlock, "", locked, "", "  " + m.styles.muted.Render("enter confirmar • esc voltar • q cancelar")}
	return fitViewWidth(lines, m.width)
}

func confirmationSkillNames(assessments []project.LinkAssessment, selected map[string]bool) ([]string, []string, error) {
	selection, err := project.NewSelection(selectedNames(assessments, selected))
	if err != nil {
		return nil, nil, err
	}
	changes, err := project.PlanSelection(assessments, selection)
	if err != nil {
		return nil, nil, err
	}
	installNames, removeNames := confirmationChangeNames(changes)
	return installNames, removeNames, nil
}

func confirmationChangeNames(changes []project.SelectionChange) ([]string, []string) {
	installNames := make([]string, 0, len(changes))
	removeNames := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.Action == project.ActionInstall {
			installNames = append(installNames, change.Skill.Name)
		}
		if change.Action == project.ActionRemove {
			removeNames = append(removeNames, change.Skill.Name)
		}
	}
	return installNames, removeNames
}

func formatConfirmationGroup(title, marker string, skillNames []string, style lipgloss.Style) string {
	header := "  " + style.Render(fmt.Sprintf("%s (%d)", title, len(skillNames)))
	if len(skillNames) == 0 {
		return header + "\n    nenhuma alteração"
	}
	itemPrefix := "    " + marker + " "
	return header + "\n" + itemPrefix + strings.Join(skillNames, "\n"+itemPrefix)
}

func fitViewWidth(lines []string, width int) string {
	fitted := make([]string, 0, len(lines))
	for _, section := range lines {
		for _, line := range strings.Split(section, "\n") {
			fitted = append(fitted, ansi.Truncate(line, maxWidth(width), "…"))
		}
	}
	return strings.Join(fitted, "\n")
}

func (m Model) Selection() (project.Selection, error) {
	if m.phase == phaseCanceled {
		return project.Selection{}, context.Canceled
	}
	return project.NewSelection(selectedNames(m.assessments, m.selected))
}

func (m Model) SetSummary(result project.BatchResult) Model {
	m.result = result
	m.phase = phaseSummary
	return m
}

type applyResultMsg struct {
	result project.BatchResult
	err    error
}

func (m Model) applySelection(selection project.Selection) tea.Cmd {
	return func() tea.Msg {
		result, err := m.apply(m.runContext, selection)
		return applyResultMsg{result: result, err: err}
	}
}

func (m Model) finishApplying(result applyResultMsg) (tea.Model, tea.Cmd) {
	if result.err != nil {
		m.resultErr = result.err
		m.phase = phaseError
		return m, nil
	}
	m = m.SetSummary(result.result)
	m.hasApplied = true
	m.assessments = project.ApplyOperationResults(m.assessments, result.result.Results)
	m.list.ResetFilter()
	_ = m.list.SetItems(assessmentItems(m.assessments))
	if m.quitAfterApply {
		return m, tea.Quit
	}
	return m, nil
}

type SelectionUI struct {
	input  io.Reader
	output io.Writer
}

func NewSelectionUI(input io.Reader, output io.Writer) SelectionUI {
	if input == nil {
		input = os.Stdin
	}
	if output == nil {
		output = os.Stdout
	}
	return SelectionUI{input: input, output: output}
}

func (ui SelectionUI) Choose(ctx context.Context, assessments []project.LinkAssessment) (project.Selection, error) {
	finalModel, err := ui.run(ctx, NewModel(assessments))
	if err != nil {
		return project.Selection{}, err
	}
	return finalModel.Selection()
}

func (ui SelectionUI) ChooseAndApply(ctx context.Context, assessments []project.LinkAssessment, apply func(context.Context, project.Selection) (project.BatchResult, error)) (project.BatchResult, error) {
	model := NewModel(assessments)
	model.runContext = ctx
	model.apply = apply
	finalModel, err := ui.run(ctx, model)
	if err != nil {
		return project.BatchResult{}, err
	}
	if finalModel.resultErr != nil {
		return finalModel.result, finalModel.resultErr
	}
	if finalModel.phase == phaseCanceled && !finalModel.hasApplied {
		return project.BatchResult{}, context.Canceled
	}
	return finalModel.result, nil
}

func (ui SelectionUI) run(ctx context.Context, model Model) (Model, error) {
	program := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(ui.input), tea.WithOutput(ui.output))
	final, err := program.Run()
	if err != nil {
		if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
			return Model{}, ctx.Err()
		}
		return Model{}, err
	}
	finalModel, ok := final.(Model)
	if !ok {
		return Model{}, fmt.Errorf("modelo final inválido %T: esperado tui.Model", final)
	}
	return finalModel, nil
}

func assessmentItems(assessments []project.LinkAssessment) []list.Item {
	items := make([]list.Item, 0, len(assessments))
	for _, assessment := range assessments {
		items = append(items, skillItem{assessment: assessment})
	}
	return items
}

func initialSelection(assessments []project.LinkAssessment) map[string]bool {
	selected := make(map[string]bool, len(assessments))
	for _, assessment := range assessments {
		selected[assessment.Skill.Name] = assessment.State == project.LinkInstalled
	}
	return selected
}

func selectedNames(assessments []project.LinkAssessment, selected map[string]bool) []string {
	names := make([]string, 0, len(assessments))
	for _, assessment := range assessments {
		if selected[assessment.Skill.Name] {
			names = append(names, assessment.Skill.Name)
		}
	}
	return names
}

func isMutable(state project.LinkState) bool {
	return state == project.LinkAbsent || state == project.LinkInstalled
}

func countLocked(assessments []project.LinkAssessment) int {
	count := 0
	for _, assessment := range assessments {
		if !isMutable(assessment.State) {
			count++
		}
	}
	return count
}
