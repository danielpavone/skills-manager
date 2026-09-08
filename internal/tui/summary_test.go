package tui

import (
	"strings"
	"testing"

	"github.com/danielpavone/skills-manager/internal/project"
)

func TestRenderSummaryGroupsOutcomesAndPreservesFailureGuidance(t *testing.T) {
	result := project.BatchResult{Results: []project.OperationResult{
		{SkillName: "install", Outcome: project.OutcomeInstalled},
		{SkillName: "remove", Outcome: project.OutcomeRemoved},
		{SkillName: "keep", Outcome: project.OutcomeUnchanged},
		{SkillName: "broken", Outcome: project.OutcomeFailed, Message: "permissão negada; verifique o acesso ao diretório"},
	}}
	view := RenderSummary(result)
	for _, expected := range []string{"instaladas:\n- install", "removidas:\n- remove", "inalteradas:\n- keep", "falhas:\n- broken: permissão negada; verifique o acesso ao diretório"} {
		assertContains(t, view, expected)
	}
	if strings.Index(view, "instaladas:") > strings.Index(view, "removidas:") {
		t.Fatal("summary groups are not in deterministic order")
	}
}

func TestRenderSummaryReportsEmptyBatch(t *testing.T) {
	if got := RenderSummary(project.BatchResult{}); got != "resumo: nenhuma operação" {
		t.Fatalf("RenderSummary() = %q", got)
	}
}
