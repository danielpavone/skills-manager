package app

import (
	"context"

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
	configured, err := config.NewCatalogConfig(catalogPath)
	if err != nil {
		return config.CatalogConfig{}, err
	}
	if _, err := c.reader.Read(ctx, configured.CatalogPath); err != nil {
		return config.CatalogConfig{}, err
	}
	if err := c.store.Save(ctx, configured); err != nil {
		return config.CatalogConfig{}, err
	}
	return configured, nil
}

func (c CatalogConfigurator) Show(ctx context.Context) (config.CatalogConfig, error) {
	return c.store.Load(ctx)
}
