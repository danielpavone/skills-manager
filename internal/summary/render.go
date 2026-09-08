package summary

import (
	"fmt"
	"strings"

	"github.com/danielpavone/skills-manager/internal/project"
)

var summaryGroups = []struct {
	outcome project.OperationOutcome
	label   string
}{
	{project.OutcomeInstalled, "instaladas"},
	{project.OutcomeRemoved, "removidas"},
	{project.OutcomeUnchanged, "inalteradas"},
	{project.OutcomeFailed, "falhas"},
}

func RenderSummary(result project.BatchResult) string {
	sections := make([]string, 0, len(summaryGroups))
	for _, group := range summaryGroups {
		items := summaryItems(result.Results, group.outcome)
		if len(items) == 0 {
			continue
		}
		sections = append(sections, renderSummaryGroup(group.label, items))
	}
	if len(sections) == 0 {
		return "resumo: nenhuma operação"
	}
	return strings.Join(sections, "\n")
}

func summaryItems(results []project.OperationResult, outcome project.OperationOutcome) []project.OperationResult {
	items := make([]project.OperationResult, 0, len(results))
	for _, result := range results {
		if result.Outcome == outcome {
			items = append(items, result)
		}
	}
	return items
}

func renderSummaryGroup(label string, items []project.OperationResult) string {
	lines := []string{label + ":"}
	for _, item := range items {
		lines = append(lines, formatSummaryItem(item))
	}
	return strings.Join(lines, "\n")
}

func formatSummaryItem(item project.OperationResult) string {
	if item.Message == "" {
		return fmt.Sprintf("- %s", item.SkillName)
	}
	return fmt.Sprintf("- %s: %s", item.SkillName, item.Message)
}
