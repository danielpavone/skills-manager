package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/danielpavone/skills-manager/internal/agentdir"
	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/config"
)

type FakeConfigStore struct {
	loaded   config.CatalogConfig
	loadErr  error
	saved    config.CatalogConfig
	saveErr  error
	saveCall int
}

func (f *FakeConfigStore) Load(context.Context) (config.CatalogConfig, error) {
	return f.loaded, f.loadErr
}

func (f *FakeConfigStore) Save(_ context.Context, configured config.CatalogConfig) error {
	f.saveCall++
	f.saved = configured
	return f.saveErr
}

type FakeCatalogReader struct {
	skills           []catalog.Skill
	err              error
	path             string
	paths            []string
	validCatalogPath string
}

func (f *FakeCatalogReader) Read(_ context.Context, catalogPath string) ([]catalog.Skill, error) {
	f.path = catalogPath
	f.paths = append(f.paths, catalogPath)
	if f.validCatalogPath != "" && catalogPath != f.validCatalogPath {
		return nil, errors.New("catalog invalid")
	}
	return f.skills, f.err
}

func TestCatalogConfiguratorSetValidatesReadsAndSaves(t *testing.T) {
	directPath := filepath.Join(t.TempDir(), "catalog")
	store := &FakeConfigStore{}
	reader := &FakeCatalogReader{skills: []catalog.Skill{{Name: "tdd"}}, validCatalogPath: directPath}
	configurator := NewCatalogConfigurator(store, reader)
	configured, err := configurator.Set(context.Background(), directPath)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if store.saveCall != 1 || store.saved != configured {
		t.Fatalf("saved = %#v, configured = %#v, calls = %d", store.saved, configured, store.saveCall)
	}
	if reader.path != configured.CatalogPath {
		t.Fatalf("reader path = %q, want %q", reader.path, configured.CatalogPath)
	}
}

func TestCatalogConfiguratorSetDoesNotSaveInvalidCatalog(t *testing.T) {
	store := &FakeConfigStore{}
	reader := &FakeCatalogReader{err: errors.New("catalog invalid")}
	configurator := NewCatalogConfigurator(store, reader)
	_, err := configurator.Set(context.Background(), "/catalog")
	if err == nil {
		t.Fatal("Set() error = nil, want reader error")
	}
	if store.saveCall != 0 {
		t.Fatalf("Save() calls = %d, want 0", store.saveCall)
	}
}

func TestCatalogConfiguratorSetFindsAgentsSkillsBelowGivenDirectory(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "skills-repository")
	catalogPath := filepath.Join(rootPath, ".agents", "skills")
	store := &FakeConfigStore{}
	reader := &FakeCatalogReader{skills: []catalog.Skill{{Name: "tdd"}}, validCatalogPath: catalogPath}
	configured, err := NewCatalogConfigurator(store, reader).Set(context.Background(), rootPath)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if configured.CatalogPath != catalogPath || store.saved != configured {
		t.Fatalf("configured = %#v, saved = %#v, want catalog path %q", configured, store.saved, catalogPath)
	}
	if len(reader.paths) != 1 || reader.paths[0] != catalogPath {
		t.Fatalf("reader paths = %#v, want nested catalog first", reader.paths)
	}
}

func TestCatalogConfigCandidatesDoesNotAppendToExplicitAgentsSkillsPath(t *testing.T) {
	catalogPath := filepath.Join(t.TempDir(), ".agents", "skills")
	candidates, err := catalogConfigCandidates(catalogPath, agentdir.Agents)
	if err != nil {
		t.Fatalf("catalogConfigCandidates() error = %v", err)
	}
	if len(candidates) != 1 || candidates[0].CatalogPath != catalogPath {
		t.Fatalf("candidates = %#v, want only %q", candidates, catalogPath)
	}
}

func TestCatalogConfiguratorSetFindsClaudeSkillsForConfiguredTarget(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "skills-repository")
	catalogPath := filepath.Join(rootPath, ".claude", "skills")
	store := &FakeConfigStore{}
	reader := &FakeCatalogReader{skills: []catalog.Skill{{Name: "tdd"}}, validCatalogPath: catalogPath}

	configured, err := NewCatalogConfigurator(store, reader).SetForTarget(context.Background(), rootPath, agentdir.Claude)
	if err != nil {
		t.Fatalf("SetForTarget() error = %v", err)
	}
	if configured.CatalogPath != catalogPath || configured.TargetDirectory != agentdir.Claude {
		t.Fatalf("configured = %#v, want Claude catalog %q", configured, catalogPath)
	}
}

func TestCatalogConfiguratorShowLoadsConfiguration(t *testing.T) {
	want := config.CatalogConfig{SchemaVersion: config.CurrentSchemaVersion, CatalogPath: "/catalog"}
	configurator := NewCatalogConfigurator(&FakeConfigStore{loaded: want}, &FakeCatalogReader{})
	got, err := configurator.Show(context.Background())
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if got != want {
		t.Fatalf("Show() = %#v, want %#v", got, want)
	}
}
