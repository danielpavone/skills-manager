package project

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/danielpavone/skills-manager/internal/catalog"
)

func TestPlanInstallsMultipleSelectedAbsentSkillsInCatalogOrder(t *testing.T) {
	assessments := planAssessments(t, LinkAbsent, LinkAbsent, LinkAbsent)
	selection, err := NewSelection([]string{"first", "third"})
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	changes, err := Plan(assessments, selection)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if got := changeActions(changes); !reflect.DeepEqual(got, []ChangeAction{ActionInstall, ActionKeep, ActionInstall}) {
		t.Fatalf("actions = %#v, want install/keep/install", got)
	}
	if got := changeNames(changes); !reflect.DeepEqual(got, []string{"first", "second", "third"}) {
		t.Fatalf("names = %#v, want catalog order", got)
	}
}

func TestPlanIsIdempotentWhenSelectionMatchesInstalledState(t *testing.T) {
	assessments := planAssessments(t, LinkInstalled, LinkAbsent)
	selection, err := NewSelection([]string{"first"})
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	changes, err := PlanSelection(assessments, selection)
	if err != nil {
		t.Fatalf("PlanSelection() error = %v", err)
	}
	if got := changeActions(changes); !reflect.DeepEqual(got, []ChangeAction{ActionKeep, ActionKeep}) {
		t.Fatalf("actions = %#v, want keep/keep", got)
	}
}

func TestPlanRemovesOnlyCorrectLinksThatWereDeselected(t *testing.T) {
	assessments := planAssessments(t, LinkInstalled, LinkBroken, LinkWrongTarget, LinkConflict, LinkAbsent)
	selection, err := NewSelection(nil)
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	changes, err := Plan(assessments, selection)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if got := changeActions(changes); !reflect.DeepEqual(got, []ChangeAction{ActionRemove, ActionKeep, ActionKeep, ActionKeep, ActionKeep}) {
		t.Fatalf("actions = %#v, want remove followed by keep actions", got)
	}
}

func TestPlanPreservesConflictsEvenWhenSelected(t *testing.T) {
	assessments := planAssessments(t, LinkBroken, LinkWrongTarget, LinkConflict)
	selection, err := NewSelection([]string{"first", "second", "third"})
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	changes, err := Plan(assessments, selection)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if got := changeActions(changes); !reflect.DeepEqual(got, []ChangeAction{ActionKeep, ActionKeep, ActionKeep}) {
		t.Fatalf("actions = %#v, want only keep actions", got)
	}
}

func TestSelectionAndPlanRejectInvalidNamesAndDestinations(t *testing.T) {
	if _, err := NewSelection([]string{"../escape"}); err == nil || !strings.Contains(err.Error(), "../escape") {
		t.Fatalf("NewSelection() error = %v, want offending name", err)
	}
	assessments := planAssessments(t, LinkAbsent)
	assessments[0].LinkPath = filepath.Join(t.TempDir(), "escape")
	selection, err := NewSelection([]string{"first"})
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	if _, err := Plan(assessments, selection); err == nil || !strings.Contains(err.Error(), assessments[0].LinkPath) {
		t.Fatalf("Plan() error = %v, want offending destination", err)
	}
}

func planAssessments(t *testing.T, states ...LinkState) []LinkAssessment {
	t.Helper()
	projectRoot := t.TempDir()
	assessments := make([]LinkAssessment, 0, len(states))
	for index, state := range states {
		name := []string{"first", "second", "third", "fourth", "fifth"}[index]
		skill := catalog.Skill{Name: name, SourcePath: filepath.Join(t.TempDir(), name)}
		assessments = append(assessments, LinkAssessment{
			Skill: skill, LinkPath: filepath.Join(projectRoot, ".agents", "skills", name), State: state,
		})
	}
	return assessments
}

func changeActions(changes []SelectionChange) []ChangeAction {
	actions := make([]ChangeAction, 0, len(changes))
	for _, change := range changes {
		actions = append(actions, change.Action)
	}
	return actions
}

func changeNames(changes []SelectionChange) []string {
	names := make([]string, 0, len(changes))
	for _, change := range changes {
		names = append(names, change.Skill.Name)
	}
	return names
}
