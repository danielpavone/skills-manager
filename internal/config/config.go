package config

import (
	"fmt"
	"path/filepath"
)

const CurrentSchemaVersion = 1

type CatalogConfig struct {
	SchemaVersion int    `json:"schema_version"`
	CatalogPath   string `json:"catalog_path"`
}

type ErrorCode string

const (
	ErrorConfigMissing ErrorCode = "config_missing"
	ErrorConfigInvalid ErrorCode = "config_invalid"
)

type DomainError struct {
	Code     ErrorCode
	Value    string
	Expected string
	Cause    error
}

func (e *DomainError) Error() string {
	message := fmt.Sprintf("%s %q: esperado %s", e.Code, e.Value, e.Expected)
	if e.Cause == nil {
		return message
	}
	return fmt.Sprintf("%s: %v", message, e.Cause)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

func NewCatalogConfig(catalogPath string) (CatalogConfig, error) {
	if catalogPath == "" {
		return CatalogConfig{}, invalidConfig(catalogPath, "caminho para um diretório de catálogo", nil)
	}
	absolutePath, err := filepath.Abs(catalogPath)
	if err != nil {
		return CatalogConfig{}, invalidConfig(catalogPath, "caminho absoluto para um diretório de catálogo", err)
	}
	configured := CatalogConfig{
		SchemaVersion: CurrentSchemaVersion,
		CatalogPath:   filepath.Clean(absolutePath),
	}
	if err := configured.Validate(); err != nil {
		return CatalogConfig{}, err
	}
	return configured, nil
}

func (c CatalogConfig) Validate() error {
	if c.SchemaVersion != CurrentSchemaVersion {
		return invalidConfig(fmt.Sprint(c.SchemaVersion), "schema_version 1", nil)
	}
	if c.CatalogPath == "" || !filepath.IsAbs(c.CatalogPath) {
		return invalidConfig(c.CatalogPath, "caminho absoluto para um diretório de catálogo", nil)
	}
	if filepath.Clean(c.CatalogPath) != c.CatalogPath {
		return invalidConfig(c.CatalogPath, "caminho absoluto e limpo para um diretório de catálogo", nil)
	}
	return nil
}

func invalidConfig(value, expected string, cause error) error {
	return &DomainError{Code: ErrorConfigInvalid, Value: value, Expected: expected, Cause: cause}
}
