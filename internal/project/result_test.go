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
