package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCatalogConfigNormalizesAbsolutePath(t *testing.T) {
	configured, err := NewCatalogConfig(filepath.Join(".", "catalog"))
	if err != nil {
		t.Fatalf("NewCatalogConfig() error = %v", err)
	}
	if !filepath.IsAbs(configured.CatalogPath) {
		t.Fatalf("CatalogPath = %q, want absolute path", configured.CatalogPath)
	}
	if configured.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", configured.SchemaVersion, CurrentSchemaVersion)
	}
}

func TestNewCatalogConfigRejectsEmptyPath(t *testing.T) {
	_, err := NewCatalogConfig("")
	assertConfigError(t, err, ErrorConfigInvalid, "caminho")
}

func TestCatalogConfigValidateRejectsUnsupportedSchema(t *testing.T) {
	err := (CatalogConfig{SchemaVersion: 99, CatalogPath: "/tmp/catalog"}).Validate()
	assertConfigError(t, err, ErrorConfigInvalid, "schema_version 1")
}

func assertConfigError(t *testing.T, err error, code ErrorCode, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = %v, want a config error", err)
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != code {
		t.Fatalf("error = %v, want code %q", err, code)
	}
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("error = %q, want %q", err.Error(), expected)
	}
}
