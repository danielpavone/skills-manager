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
	phaseSummary  phase = "summary"
	phaseCanceled phase = "canceled"
)

type Model struct {
	assessments []project.LinkAssessment
	selected    map[string]bool
	list        list.Model
	styles      viewStyles
	phase       phase
	width       int
	height      int
	result      project.BatchResult
}

func NewModel(assessments []project.LinkAssessment) Model {
	selected := initialSelection(assessments)
	items := assessmentItems(assessments)
	styles := newViewStyles()
	delegate := newSkillDelegate(selected, styles)
	itemsList := list.New(items, delegate, 80, 12)
	itemsList.Title = "skills"
	itemsList.FilterInput.Prompt = "pesquisa: "
	itemsList.SetShowTitle(false)
	itemsList.SetShowStatusBar(false)
	itemsList.SetShowHelp(false)
	itemsList.SetShowFilter(false)
	itemsList.DisableQuitKeybindings()
	return Model{
		assessments: append([]project.LinkAssessment(nil), assessments...),
		selected:    selected,
		list:        itemsList,
		styles:      styles,
		phase:       phaseList,
		width:       80,
		height:      20,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.resize(size.Width, size.Height)
		return m, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m.updateListMessage(msg)
	}
	if m.phase == phaseList {
		return m.updateList(key)
	}
	if m.phase == phaseConfirm {
		return m.updateConfirmation(key)
	}
	if m.phase == phaseSummary && isCancelKey(key) {
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
	if isKey(key, "ctrl+c") {
		m.phase = phaseCanceled
		return m, tea.Quit
	}
	if m.list.FilterState() != list.Filtering && isKey(key, "space") {
		m.toggleCurrent()
		return m, nil
	}
	if m.list.FilterState() != list.Filtering && isKey(key, "enter") {
		m.phase = phaseConfirm
		return m, nil
	}
	if m.list.FilterState() != list.Filtering && isCancelKey(key) {
		m.phase = phaseCanceled
		return m, tea.Quit
	}
	return m.updateListMessage(key)
}

func (m Model) updateConfirmation(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if isConfirmKey(key) {
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
	case phaseSummary:
		return RenderSummary(m.result)
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
	model := NewModel(assessments)
	program := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(ui.input), tea.WithOutput(ui.output))
	final, err := program.Run()
	if err != nil {
		if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
			return project.Selection{}, ctx.Err()
		}
		return project.Selection{}, err
	}
	finalModel, ok := final.(Model)
	if !ok {
		return project.Selection{}, fmt.Errorf("modelo final inválido %T: esperado tui.Model", final)
	}
	return finalModel.Selection()
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
