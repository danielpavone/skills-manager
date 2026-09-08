package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/danielpavone/skills-manager/internal/project"
)

func (m Model) renderResult() string {
	title := "operação concluída com sucesso"
	if m.result.HasFailures {
		title = "operação concluída com falhas"
	}
	lines := []string{"", "  " + m.styles.title.Render(title), "  " + m.styles.subtitle.Render(resultSubtitle(m.result)), ""}
	lines = append(lines, resultSections(m.result, m.styles)...)
	lines = append(lines, "", "  "+m.styles.muted.Render("enter voltar às skills • q/esc sair"))
	return fitViewWidth(lines, m.width)
}

func resultSubtitle(result project.BatchResult) string {
	if result.HasFailures {
		return "Revise as skills que não puderam ser atualizadas"
	}
	return "As skills deste projeto estão atualizadas"
}

func resultSections(result project.BatchResult, styles viewStyles) []string {
	sections := make([]string, 0, 7)
	sections = appendResultGroup(sections, "Instaladas", "+", outcomeResults(result, project.OutcomeInstalled), styles.install)
	sections = appendResultGroup(sections, "Removidas", "-", outcomeResults(result, project.OutcomeRemoved), styles.remove)
	sections = appendResultGroup(sections, "Conflitos preservados", "!", conflictResults(result), styles.warning)
	sections = appendResultGroup(sections, "Falhas", "!", outcomeResults(result, project.OutcomeFailed), styles.remove)
	if len(sections) == 0 {
		return []string{"  " + styles.muted.Render("Nenhuma alteração foi necessária.")}
	}
	return sections
}

func appendResultGroup(sections []string, title, marker string, results []project.OperationResult, style lipgloss.Style) []string {
	if len(results) == 0 {
		return sections
	}
	if len(sections) > 0 {
		sections = append(sections, "")
	}
	return append(sections, formatResultGroup(title, marker, results, style))
}

func formatResultGroup(title, marker string, results []project.OperationResult, style lipgloss.Style) string {
	header := "  " + style.Render(fmt.Sprintf("%s (%d)", title, len(results)))
	items := make([]string, 0, len(results))
	for _, result := range results {
		items = append(items, "    "+marker+" "+formatResultItem(result))
	}
	return header + "\n" + strings.Join(items, "\n")
}

func formatResultItem(result project.OperationResult) string {
	if result.Message == "" {
		return result.SkillName
	}
	return result.SkillName + ": " + result.Message
}

func outcomeResults(result project.BatchResult, outcome project.OperationOutcome) []project.OperationResult {
	matches := make([]project.OperationResult, 0, len(result.Results))
	for _, operation := range result.Results {
		if operation.Outcome == outcome {
			matches = append(matches, operation)
		}
	}
	return matches
}

func conflictResults(result project.BatchResult) []project.OperationResult {
	conflicts := make([]project.OperationResult, 0, len(result.Results))
	for _, operation := range result.Results {
		if operation.Outcome == project.OutcomeUnchanged && operation.ErrorCode == project.ErrorLinkConflict {
			conflicts = append(conflicts, operation)
		}
	}
	return conflicts
}
