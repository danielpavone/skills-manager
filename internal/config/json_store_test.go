package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/danielpavone/skills-manager/internal/agentdir"
)

func TestJSONStorePersistsAndReloadsConfiguration(t *testing.T) {
	directory := t.TempDir()
	store := NewJSONStore(filepath.Join(directory, "settings"))
	configured := CatalogConfig{SchemaVersion: CurrentSchemaVersion, CatalogPath: filepath.Join(directory, "catalog")}
	if err := store.Save(context.Background(), configured); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	reloaded, err := NewJSONStore(filepath.Join(directory, "settings")).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reloaded != configured {
		t.Fatalf("reloaded = %#v, want %#v", reloaded, configured)
	}
	if mode := fileMode(t, filepath.Join(directory, "settings", configFileName)); runtime.GOOS != "windows" && mode.Perm() != 0600 {
		t.Fatalf("config mode = %o, want 600", mode.Perm())
	}
}

func TestJSONStoreLoadsLegacyConfigurationWithAgentsDefault(t *testing.T) {
	directory := t.TempDir()
	legacy := `{"schema_version":1,"catalog_path":"/catalog"}`
	if err := os.WriteFile(filepath.Join(directory, configFileName), []byte(legacy), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	configured, err := NewJSONStore(directory).Load(context.Background())
	if err != nil || configured.EffectiveTargetDirectory() != agentdir.Agents {
		t.Fatalf("Load() = %#v, %v, want legacy .agents default", configured, err)
	}
}

func TestJSONStoreReportsMissingConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings")
	_, err := NewJSONStore(path).Load(context.Background())
	assertConfigError(t, err, ErrorConfigMissing, path)
	if !strings.Contains(err.Error(), "config set") {
		t.Fatalf("error = %q, want setup guidance", err.Error())
	}
}

func TestJSONStoreRejectsInvalidContentWithoutChangingPreviousFile(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "settings")
	store := NewJSONStore(directory)
	configured := CatalogConfig{SchemaVersion: CurrentSchemaVersion, CatalogPath: filepath.Join(t.TempDir(), "original")}
	if err := store.Save(context.Background(), configured); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, configFileName), []byte(`{"schema_version": 99}`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := store.Load(context.Background())
	assertConfigError(t, err, ErrorConfigInvalid, "schema_version")
}

func TestJSONStorePreservesPreviousFileWhenReplacementFails(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "settings")
	store := NewJSONStore(directory)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	target := filepath.Join(directory, configFileName)
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", target, err)
	}
	configured := CatalogConfig{SchemaVersion: CurrentSchemaVersion, CatalogPath: filepath.Join(t.TempDir(), "catalog")}
	if err := store.Save(context.Background(), configured); err == nil {
		t.Fatal("Save() error = nil, want replacement failure")
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("previous target = (%v, %v), want existing directory", info, err)
	}
}

func TestNewUserJSONStoreUsesSkillsManagerDirectory(t *testing.T) {
	store, err := NewUserJSONStore()
	if err != nil {
		t.Fatalf("NewUserJSONStore() error = %v", err)
	}
	if filepath.Base(store.directory) != "skills-manager" {
		t.Fatalf("directory = %q, want skills-manager suffix", store.directory)
	}
}

func TestJSONStoreRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := NewJSONStore(t.TempDir())
	if _, err := store.Load(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want context canceled", err)
	}
	configured := CatalogConfig{SchemaVersion: CurrentSchemaVersion, CatalogPath: "/catalog"}
	if err := store.Save(ctx, configured); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save() error = %v, want context canceled", err)
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	return info.Mode()
}
