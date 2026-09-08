package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

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
	skills []catalog.Skill
	err    error
	path   string
}

func (f *FakeCatalogReader) Read(_ context.Context, catalogPath string) ([]catalog.Skill, error) {
	f.path = catalogPath
	return f.skills, f.err
}

func TestCatalogConfiguratorSetValidatesReadsAndSaves(t *testing.T) {
	store := &FakeConfigStore{}
	reader := &FakeCatalogReader{skills: []catalog.Skill{{Name: "tdd"}}}
	configurator := NewCatalogConfigurator(store, reader)
	configured, err := configurator.Set(context.Background(), filepath.Join(".", "catalog"))
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
