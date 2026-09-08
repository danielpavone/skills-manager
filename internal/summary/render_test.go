package summary

import (
	"strings"
	"testing"

	"github.com/danielpavone/skills-manager/internal/project"
)

func TestSharedSummaryIncludesEveryOutcomeAndFailureMessage(t *testing.T) {
	batch := project.BatchResult{Results: []project.OperationResult{
		{SkillName: "a", Outcome: project.OutcomeInstalled},
		{SkillName: "b", Outcome: project.OutcomeRemoved},
		{SkillName: "c", Outcome: project.OutcomeUnchanged},
		{SkillName: "d", Outcome: project.OutcomeFailed, Message: "permissão negada"},
	}}
	actual := RenderSummary(batch)
	for _, expected := range []string{"instaladas:\n- a", "removidas:\n- b", "inalteradas:\n- c", "falhas:\n- d: permissão negada"} {
		if !strings.Contains(actual, expected) {
			t.Fatalf("summary = %q, want %q", actual, expected)
		}
	}
}

func TestSharedSummaryReportsEmptyBatch(t *testing.T) {
	actual := RenderSummary(project.BatchResult{})
	if actual != "resumo: nenhuma operação" {
		t.Fatalf("summary = %q, want empty operation message", actual)
	}
}
