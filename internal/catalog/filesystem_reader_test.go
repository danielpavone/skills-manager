package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFilesystemReaderDiscoversValidSkillsInDeterministicOrder(t *testing.T) {
	root := t.TempDir()
	createSkill(t, root, "beta")
	createSkill(t, root, "Alpha")
	createSkill(t, root, "gamma")
	if err := os.WriteFile(filepath.Join(root, "not-a-skill"), []byte("x"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "missing-manifest"), 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	skills, err := NewFilesystemReader().Read(context.Background(), root)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	wantNames := []string{"Alpha", "beta", "gamma"}
	gotNames := []string{skills[0].Name, skills[1].Name, skills[2].Name}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("skill names = %#v, want %#v", gotNames, wantNames)
	}
	if !filepath.IsAbs(skills[0].SourcePath) {
		t.Fatalf("SourcePath = %q, want absolute path", skills[0].SourcePath)
	}
}

func TestFilesystemReaderRejectsInvalidCatalogs(t *testing.T) {
	tests := map[string]string{
		"missing": filepath.Join(t.TempDir(), "missing"),
		"empty":   t.TempDir(),
		"file":    filepath.Join(t.TempDir(), "catalog-file"),
	}
	if err := os.WriteFile(tests["file"], []byte("not a directory"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	for name, path := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewFilesystemReader().Read(context.Background(), path)
			assertCatalogError(t, err, path)
		})
	}
}

func TestFilesystemReaderHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewFilesystemReader().Read(ctx, t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Read() error = %v, want context canceled", err)
	}
}

func createSkill(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("# "+name), 0600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", name, err)
	}
}

func assertCatalogError(t *testing.T, err error, path string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want catalog error for %q", path)
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrorCatalogInvalid {
		t.Fatalf("error = %v, want catalog_invalid", err)
	}
	if !containsCatalogText(err.Error(), path) || !containsCatalogText(err.Error(), "esperado") {
		t.Fatalf("error = %q, want path and expected shape", err.Error())
	}
}

func containsCatalogText(value, expected string) bool {
	for index := 0; index+len(expected) <= len(value); index++ {
		if value[index:index+len(expected)] == expected {
			return true
		}
	}
	return false
}
