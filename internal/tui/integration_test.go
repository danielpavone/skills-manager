package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielpavone/skills-manager/internal/app"
	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/config"
	"github.com/danielpavone/skills-manager/internal/project"
)

type fixedConfigStore struct {
	configured config.CatalogConfig
}

func (s fixedConfigStore) Load(context.Context) (config.CatalogConfig, error) {
	return s.configured, nil
}

func (s fixedConfigStore) Save(context.Context, config.CatalogConfig) error { return nil }

type fixedCatalogReader struct {
	skills []catalog.Skill
}

func (r fixedCatalogReader) Read(context.Context, string) ([]catalog.Skill, error) {
	return append([]catalog.Skill(nil), r.skills...), nil
}

func TestSelectionUIInstallsSelectedSkillThroughProjectWorkflow(t *testing.T) {
	projectPath, skill := testProjectAndSkill(t, "install")
	result := runWorkflow(t, projectPath, skill, " \r\rq")
	if result.Results[0].Outcome != project.OutcomeInstalled {
		t.Fatalf("outcome = %q, want installed", result.Results[0].Outcome)
	}
	linkPath := filepath.Join(projectPath, ".agents", "skills", skill.Name)
	if target, err := os.Readlink(linkPath); err != nil || target != skill.SourcePath {
		t.Fatalf("link = %q, %v; want %q", target, err, skill.SourcePath)
	}
}

func TestSelectionUIReopensAndRemovesOnlyTheLocalLink(t *testing.T) {
	projectPath, skill := testProjectAndSkill(t, "remove")
	runWorkflow(t, projectPath, skill, " \r\rq")
	result := runWorkflow(t, projectPath, skill, " \r\rq")
	if result.Results[0].Outcome != project.OutcomeRemoved {
		t.Fatalf("outcome = %q, want removed", result.Results[0].Outcome)
	}
	if _, err := os.Stat(skill.SourcePath); err != nil {
		t.Fatalf("source stat error = %v, want source preserved", err)
	}
}

func runWorkflow(t *testing.T, projectPath string, skill catalog.Skill, input string) project.BatchResult {
	t.Helper()
	store := fixedConfigStore{configured: config.CatalogConfig{CatalogPath: filepath.Dir(skill.SourcePath), SchemaVersion: config.CurrentSchemaVersion}}
	reader := fixedCatalogReader{skills: []catalog.Skill{skill}}
	selection := NewSelectionUI(strings.NewReader(input), nil)
	manager := app.NewManageProject(store, reader, project.NewFilesystemLinks(), selection)
	result, err := manager.Manage(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("Manage() error = %v", err)
	}
	return result
}

func testProjectAndSkill(t *testing.T, name string) (string, catalog.Skill) {
	t.Helper()
	projectPath := t.TempDir()
	catalogPath := t.TempDir()
	sourcePath := filepath.Join(catalogPath, name)
	if err := os.MkdirAll(sourcePath, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("# "+name), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return projectPath, catalog.Skill{Name: name, SourcePath: sourcePath}
}
