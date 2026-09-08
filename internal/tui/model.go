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
	delegate := newSkillDelegate(selected)
	itemsList := list.New(items, delegate, 80, 12)
	itemsList.Title = "skills"
	itemsList.SetShowTitle(false)
	itemsList.SetShowStatusBar(false)
	itemsList.SetShowHelp(false)
	itemsList.SetShowFilter(false)
	itemsList.DisableQuitKeybindings()
	return Model{
		assessments: append([]project.LinkAssessment(nil), assessments...),
		selected:    selected,
		list:        itemsList,
		styles:      newViewStyles(),
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
		return m, nil
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
	updated, cmd := m.list.Update(key)
	m.list = updated
	return m, cmd
}

func (m Model) updateConfirmation(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if isConfirmKey(key) && key.String() == "y" {
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
	m.list.SetSize(m.width, maxWidth(m.height-6))
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
	title := m.styles.title.Render("skills-manager — catálogo")
	search := "pesquisa: " + m.list.FilterValue()
	help := m.styles.muted.Render("/ pesquisar • ↑/↓ ou j/k mover • space selecionar • enter confirmar • esc/q cancelar")
	return strings.Join([]string{title, search, m.list.View(), help}, "\n")
}

func (m Model) renderConfirmation() string {
	selected := countSelected(m.assessments, m.selected)
	locked := countLocked(m.assessments)
	lines := []string{
		m.styles.title.Render("confirmar alterações"),
		fmt.Sprintf("selecionadas: %d • conflitos bloqueados: %d", selected, locked),
		"y confirmar • esc voltar • q cancelar",
	}
	return strings.Join(lines, "\n")
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

func countSelected(assessments []project.LinkAssessment, selected map[string]bool) int {
	return len(selectedNames(assessments, selected))
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
