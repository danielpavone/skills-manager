package project

import (
	"path/filepath"
	"testing"

	"github.com/danielpavone/skills-manager/internal/catalog"
)

func TestOnlyKeepsRecognizesMutationPlans(t *testing.T) {
	skill := catalog.Skill{Name: "tdd", SourcePath: filepath.Join(t.TempDir(), "tdd")}
	change := SelectionChange{Skill: skill, Action: ActionKeep}
	if !OnlyKeeps([]SelectionChange{change}) {
		t.Fatal("OnlyKeeps() = false, want true")
	}
	change.Action = ActionInstall
	if OnlyKeeps([]SelectionChange{change}) {
		t.Fatal("OnlyKeeps() = true, want false")
	}
}

func TestUnchangedBatchPreservesPlanOrder(t *testing.T) {
	first := SelectionChange{Skill: catalog.Skill{Name: "a"}, Action: ActionKeep}
	second := SelectionChange{Skill: catalog.Skill{Name: "b"}, Action: ActionKeep}
	result := UnchangedBatch([]SelectionChange{first, second})
	if len(result.Results) != 2 || result.Results[0].SkillName != "a" || result.Results[1].SkillName != "b" {
		t.Fatalf("results = %#v, want a then b", result.Results)
	}
	if result.Results[0].Outcome != OutcomeUnchanged || result.HasFailures {
		t.Fatalf("result = %#v, want unchanged without failures", result)
	}
}

func TestApplyOperationResultsUpdatesOnlySuccessfulLinkStates(t *testing.T) {
	assessments := []LinkAssessment{
		{Skill: catalog.Skill{Name: "installed", SourcePath: "/catalog/installed"}, State: LinkAbsent},
		{Skill: catalog.Skill{Name: "removed", SourcePath: "/catalog/removed"}, State: LinkInstalled},
		{Skill: catalog.Skill{Name: "failed", SourcePath: "/catalog/failed"}, State: LinkAbsent},
	}
	results := []OperationResult{
		{SkillName: "installed", Outcome: OutcomeInstalled},
		{SkillName: "removed", Outcome: OutcomeRemoved},
		{SkillName: "failed", Outcome: OutcomeFailed},
	}
	updated := ApplyOperationResults(assessments, results)
	if updated[0].State != LinkInstalled || updated[0].ActualTarget == nil || *updated[0].ActualTarget != "/catalog/installed" {
		t.Fatalf("installed assessment = %#v, want installed with catalog target", updated[0])
	}
	if updated[1].State != LinkAbsent || updated[1].ActualTarget != nil {
		t.Fatalf("removed assessment = %#v, want absent without target", updated[1])
	}
	if updated[2].State != LinkAbsent {
		t.Fatalf("failed assessment = %#v, want original state", updated[2])
	}
	if assessments[0].State != LinkAbsent {
		t.Fatal("ApplyOperationResults mutated its input")
	}
}
