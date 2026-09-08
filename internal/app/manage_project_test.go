package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/config"
	"github.com/danielpavone/skills-manager/internal/project"
)

type FakeProjectLinks struct {
	assessments []project.LinkAssessment
	inspectErr  error
	result      project.BatchResult
	inspectPath string
	applyPath   string
	changes     []project.SelectionChange
	applyCalls  int
}

func (f *FakeProjectLinks) Inspect(_ context.Context, projectPath string, _ []catalog.Skill) ([]project.LinkAssessment, error) {
	f.inspectPath = projectPath
	return f.assessments, f.inspectErr
}

func (f *FakeProjectLinks) Apply(_ context.Context, projectPath string, changes []project.SelectionChange) project.BatchResult {
	f.applyCalls++
	f.applyPath = projectPath
	f.changes = append([]project.SelectionChange(nil), changes...)
	return f.result
}

type FakeSelectionUI struct {
	selection   project.Selection
	err         error
	assessments []project.LinkAssessment
	chooseCalls int
}

func (f *FakeSelectionUI) Choose(_ context.Context, assessments []project.LinkAssessment) (project.Selection, error) {
	f.chooseCalls++
	f.assessments = append([]project.LinkAssessment(nil), assessments...)
	return f.selection, f.err
}

func TestManageProjectRunsTheProjectWorkflowInOrder(t *testing.T) {
	projectPath := t.TempDir()
	skill := catalog.Skill{Name: "tdd", SourcePath: filepath.Join(t.TempDir(), "tdd")}
	linkPath := filepath.Join(projectPath, ".agents", "skills", skill.Name)
	assessment := project.LinkAssessment{Skill: skill, LinkPath: linkPath, State: project.LinkAbsent}
	store := &FakeConfigStore{loaded: config.CatalogConfig{CatalogPath: "/catalog", SchemaVersion: config.CurrentSchemaVersion}}
	reader := &FakeCatalogReader{skills: []catalog.Skill{skill}}
	links := &FakeProjectLinks{assessments: []project.LinkAssessment{assessment}, result: installedBatch(skill.Name)}
	ui := &FakeSelectionUI{selection: mustSelection(t, skill.Name)}
	useCase := NewManageProject(store, reader, links, ui)

	result, err := useCase.Manage(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("Manage() error = %v", err)
	}
	if links.inspectPath != projectPath || links.applyPath != projectPath {
		t.Fatalf("project paths = %q and %q, want %q", links.inspectPath, links.applyPath, projectPath)
	}
	if ui.chooseCalls != 1 || len(ui.assessments) != 1 || links.applyCalls != 1 {
		t.Fatalf("workflow calls = choose %d, assessments %d, apply %d", ui.chooseCalls, len(ui.assessments), links.applyCalls)
	}
	if result.Results[0].Outcome != project.OutcomeInstalled {
		t.Fatalf("result = %#v, want installed", result.Results)
	}
	if len(links.changes) != 1 || links.changes[0].Action != project.ActionInstall {
		t.Fatalf("changes = %#v, want install", links.changes)
	}
}

func TestManageProjectSkipsApplyWhenConfirmationHasNoDifferences(t *testing.T) {
	projectPath := t.TempDir()
	skill := catalog.Skill{Name: "tdd", SourcePath: filepath.Join(t.TempDir(), "tdd")}
	assessment := project.LinkAssessment{
		Skill: skill, LinkPath: filepath.Join(projectPath, ".agents", "skills", skill.Name), State: project.LinkInstalled,
	}
	links := &FakeProjectLinks{assessments: []project.LinkAssessment{assessment}}
	ui := &FakeSelectionUI{selection: mustSelection(t, skill.Name)}
	useCase := NewManageProject(
		&FakeConfigStore{loaded: config.CatalogConfig{CatalogPath: "/catalog", SchemaVersion: config.CurrentSchemaVersion}},
		&FakeCatalogReader{skills: []catalog.Skill{skill}}, links, ui,
	)

	result, err := useCase.Run(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if links.applyCalls != 0 || len(result.Results) != 1 || result.Results[0].Outcome != project.OutcomeUnchanged {
		t.Fatalf("apply calls = %d, result = %#v, want one unchanged result without apply", links.applyCalls, result)
	}
}

func TestManageProjectReturnsCancellationWithoutApplying(t *testing.T) {
	links := &FakeProjectLinks{}
	ui := &FakeSelectionUI{err: ErrSelectionCanceled}
	useCase := NewManageProject(
		&FakeConfigStore{loaded: config.CatalogConfig{CatalogPath: "/catalog", SchemaVersion: config.CurrentSchemaVersion}},
		&FakeCatalogReader{}, links, ui,
	)

	_, err := useCase.Manage(context.Background(), t.TempDir())
	if !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("Manage() error = %v, want cancellation", err)
	}
	if links.applyCalls != 0 {
		t.Fatalf("Apply() calls = %d, want 0", links.applyCalls)
	}
}

func TestCanceledSelectionUIReturnsStableCancellation(t *testing.T) {
	_, err := NewCanceledSelectionUI().Choose(context.Background(), nil)
	if !errors.Is(err, ErrSelectionCanceled) {
		t.Fatalf("Choose() error = %v, want cancellation", err)
	}
}

func mustSelection(t *testing.T, names ...string) project.Selection {
	t.Helper()
	selection, err := project.NewSelection(names)
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	return selection
}

func installedBatch(name string) project.BatchResult {
	return project.BatchResult{Results: []project.OperationResult{{
		SkillName: name, Action: project.ActionInstall, Outcome: project.OutcomeInstalled,
	}}}
}
