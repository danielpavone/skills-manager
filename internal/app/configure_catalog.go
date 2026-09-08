package app

import (
	"context"

	"github.com/danielpavone/skills-manager/internal/agentdir"
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
	return c.SetForTarget(ctx, catalogPath, agentdir.Agents)
}

func (c CatalogConfigurator) SetForTarget(ctx context.Context, catalogPath string, target agentdir.Directory) (config.CatalogConfig, error) {
	candidates, err := catalogConfigCandidates(catalogPath, target)
	if err != nil {
		return config.CatalogConfig{}, err
	}
	return c.saveFirstReadable(ctx, candidates)
}

func (c CatalogConfigurator) saveFirstReadable(ctx context.Context, candidates []config.CatalogConfig) (config.CatalogConfig, error) {
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

func catalogConfigCandidates(catalogPath string, target agentdir.Directory) ([]config.CatalogConfig, error) {
	direct, err := config.NewCatalogConfigForTarget(catalogPath, target)
	if err != nil {
		return nil, err
	}
	if _, err := agentdir.FromSkillsPath(direct.CatalogPath); err == nil {
		return []config.CatalogConfig{direct}, nil
	}
	nestedPath := agentdir.SkillsPath(direct.CatalogPath, target)
	nested, err := config.NewCatalogConfigForTarget(nestedPath, target)
	if err != nil {
		return nil, err
	}
	return []config.CatalogConfig{nested, direct}, nil
}

func (c CatalogConfigurator) Show(ctx context.Context) (config.CatalogConfig, error) {
	return c.store.Load(ctx)
}
