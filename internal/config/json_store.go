package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const configFileName = "config.json"

type JSONStore struct {
	directory string
	filePath  string
}

func NewJSONStore(directory string) JSONStore {
	cleanDirectory := filepath.Clean(directory)
	return JSONStore{
		directory: cleanDirectory,
		filePath:  filepath.Join(cleanDirectory, configFileName),
	}
}

func NewUserJSONStore() (JSONStore, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return JSONStore{}, fmt.Errorf("não foi possível localizar a configuração do usuário: %w", err)
	}
	return NewJSONStore(filepath.Join(directory, "skills-manager")), nil
}

func (s JSONStore) Load(ctx context.Context) (CatalogConfig, error) {
	if err := contextError(ctx); err != nil {
		return CatalogConfig{}, err
	}
	content, err := os.ReadFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return CatalogConfig{}, &DomainError{
			Code: ErrorConfigMissing, Value: s.filePath,
			Expected: "configuração criada por 'skills-manager config set <caminho>'",
		}
	}
	if err != nil {
		return CatalogConfig{}, invalidConfig(s.filePath, "arquivo JSON de configuração legível", err)
	}
	return decodeConfig(s.filePath, content)
}

func (s JSONStore) Save(ctx context.Context, configured CatalogConfig) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := configured.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(s.directory, 0700); err != nil {
		return invalidConfig(s.directory, "diretório de configuração gravável", err)
	}
	if err := os.Chmod(s.directory, 0700); err != nil {
		return invalidConfig(s.directory, "diretório de configuração ajustável para modo 0700", err)
	}
	content, err := json.MarshalIndent(configured, "", "  ")
	if err != nil {
		return invalidConfig(s.filePath, "configuração serializável em JSON", err)
	}
	return s.replaceAtomically(ctx, append(content, '\n'))
}

func (s JSONStore) replaceAtomically(ctx context.Context, content []byte) error {
	temporary, err := os.CreateTemp(s.directory, ".config-*.tmp")
	if err != nil {
		return invalidConfig(s.filePath, "arquivo temporário de configuração criável", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	defer temporary.Close()
	if err := writeConfigTemporary(temporary, content); err != nil {
		return err
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, s.filePath); err != nil {
		return invalidConfig(s.filePath, "arquivo de configuração substituível atomicamente", err)
	}
	return nil
}

func writeConfigTemporary(temporary *os.File, content []byte) error {
	temporaryPath := temporary.Name()
	if err := temporary.Chmod(0600); err != nil {
		return invalidConfig(temporaryPath, "arquivo temporário ajustável para modo 0600", err)
	}
	if _, err := temporary.Write(content); err != nil {
		return invalidConfig(temporaryPath, "conteúdo JSON gravável", err)
	}
	if err := temporary.Sync(); err != nil {
		return invalidConfig(temporaryPath, "arquivo temporário sincronizável", err)
	}
	if err := temporary.Close(); err != nil {
		return invalidConfig(temporaryPath, "arquivo temporário fechável", err)
	}
	return nil
}

func decodeConfig(path string, content []byte) (CatalogConfig, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var configured CatalogConfig
	if err := decoder.Decode(&configured); err != nil {
		return CatalogConfig{}, invalidConfig(path, "JSON com schema_version e catalog_path", err)
	}
	var extra map[string]json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return CatalogConfig{}, invalidConfig(path, "JSON com um único objeto de configuração", err)
	}
	if err := configured.Validate(); err != nil {
		return CatalogConfig{}, err
	}
	return configured, nil
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
