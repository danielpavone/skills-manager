package app

import (
	"context"
	"path/filepath"

	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/config"
)

type ConfigStore interface {
	Load(ctx context.Context) (config.CatalogConfig, error)
	Save(ctx context.Context, configured config.CatalogConfig) error
}

type CatalogReader interface {
	Read(ctx context.Context, catalogPath string) ([]catalog.Skill, error)
}

type CatalogConfigurator struct {
	store  ConfigStore
	reader CatalogReader
}

func NewCatalogConfigurator(store ConfigStore, reader CatalogReader) CatalogConfigurator {
	return CatalogConfigurator{store: store, reader: reader}
}

func (c CatalogConfigurator) Set(ctx context.Context, catalogPath string) (config.CatalogConfig, error) {
	candidates, err := catalogConfigCandidates(catalogPath)
	if err != nil {
		return config.CatalogConfig{}, err
	}
	var readErr error
	for _, candidate := range candidates {
		if _, readErr = c.reader.Read(ctx, candidate.CatalogPath); readErr != nil {
			continue
		}
		if err := c.store.Save(ctx, candidate); err != nil {
			return config.CatalogConfig{}, err
		}
		return candidate, nil
	}
	return config.CatalogConfig{}, readErr
}

func catalogConfigCandidates(catalogPath string) ([]config.CatalogConfig, error) {
	direct, err := config.NewCatalogConfig(catalogPath)
	if err != nil {
		return nil, err
	}
	if filepath.Base(direct.CatalogPath) == "skills" && filepath.Base(filepath.Dir(direct.CatalogPath)) == ".agents" {
		return []config.CatalogConfig{direct}, nil
	}
	nested, err := config.NewCatalogConfig(filepath.Join(direct.CatalogPath, ".agents", "skills"))
	if err != nil {
		return nil, err
	}
	return []config.CatalogConfig{nested, direct}, nil
}

func (c CatalogConfigurator) Show(ctx context.Context) (config.CatalogConfig, error) {
	return c.store.Load(ctx)
}
