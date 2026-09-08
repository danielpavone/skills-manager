package tui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/danielpavone/skills-manager/internal/project"
)

type skillItem struct {
	assessment project.LinkAssessment
}

func (i skillItem) FilterValue() string {
	return i.assessment.Skill.Name
}

type skillDelegate struct {
	selected map[string]bool
}

func newSkillDelegate(selected map[string]bool) skillDelegate {
	return skillDelegate{selected: selected}
}

func (d skillDelegate) Height() int { return 1 }

func (d skillDelegate) Spacing() int { return 0 }

func (d skillDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d skillDelegate) Render(writer io.Writer, model list.Model, index int, item list.Item) {
	skill, ok := item.(skillItem)
	if !ok {
		return
	}
	line := formatSkillLine(skill, d.selected[skill.assessment.Skill.Name], index == model.Index())
	line = ansi.Truncate(line, maxWidth(model.Width()), "…")
	_, _ = fmt.Fprint(writer, line)
}

func formatSkillLine(item skillItem, selected, focused bool) string {
	marker, state := itemMarkerAndState(item.assessment, selected)
	prefix := "  "
	if focused {
		prefix = "> "
	}
	return fmt.Sprintf("%s%s %-24s %s", prefix, marker, item.assessment.Skill.Name, state)
}

func itemMarkerAndState(assessment project.LinkAssessment, selected bool) (string, string) {
	switch assessment.State {
	case project.LinkInstalled:
		if selected {
			return "[x]", "installed / keep"
		}
		return "[ ]", "installed / remove"
	case project.LinkAbsent:
		if selected {
			return "[+]", "absent / install"
		}
		return "[ ]", "absent"
	default:
		return "[!]", string(assessment.State) + " / locked"
	}
}

func maxWidth(width int) int {
	if width < 1 {
		return 1
	}
	return width
}
