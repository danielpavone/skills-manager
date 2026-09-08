package app

import (
	"context"
	"errors"

	"github.com/danielpavone/skills-manager/internal/agentdir"
	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/project"
)

var ErrSelectionCanceled = errors.New("seleção cancelada pelo usuário")

type ProjectLinks interface {
	Inspect(ctx context.Context, projectPath string, target agentdir.Directory, skills []catalog.Skill) ([]project.LinkAssessment, error)
	Apply(ctx context.Context, projectPath string, changes []project.SelectionChange) project.BatchResult
}

type SelectionUI interface {
	Choose(ctx context.Context, assessments []project.LinkAssessment) (project.Selection, error)
}

type SelectionSessionUI interface {
	ChooseAndApply(ctx context.Context, assessments []project.LinkAssessment, apply func(context.Context, project.Selection) (project.BatchResult, error)) (project.BatchResult, error)
}

type ManageProject struct {
	store     ConfigStore
	reader    CatalogReader
	links     ProjectLinks
	selection SelectionUI
}

func NewManageProject(store ConfigStore, reader CatalogReader, links ProjectLinks, selection SelectionUI) ManageProject {
	return ManageProject{store: store, reader: reader, links: links, selection: selection}
}

func (m ManageProject) Manage(ctx context.Context, projectPath string) (project.BatchResult, error) {
	assessments, err := m.inspectProject(ctx, projectPath)
	if err != nil {
		return project.BatchResult{}, err
	}
	if session, ok := m.selection.(SelectionSessionUI); ok {
		return m.manageSession(ctx, projectPath, assessments, session)
	}
	changes, err := m.planChanges(ctx, assessments)
	if err != nil {
		return project.BatchResult{}, normalizeSelectionError(err)
	}
	return m.applySelection(ctx, projectPath, assessments, changes)
}

func (m ManageProject) manageSession(ctx context.Context, projectPath string, assessments []project.LinkAssessment, session SelectionSessionUI) (project.BatchResult, error) {
	result, err := session.ChooseAndApply(ctx, assessments, func(applyCtx context.Context, selection project.Selection) (project.BatchResult, error) {
		changes, planErr := project.Plan(assessments, selection)
		if planErr != nil {
			return project.BatchResult{}, planErr
		}
		return m.applySelection(applyCtx, projectPath, assessments, changes)
	})
	return result, normalizeSelectionError(err)
}

func (m ManageProject) applySelection(ctx context.Context, projectPath string, assessments []project.LinkAssessment, changes []project.SelectionChange) (project.BatchResult, error) {
	if project.OnlyKeeps(changes) {
		return project.DescribeConflicts(project.UnchangedBatch(changes), assessments), nil
	}
	return project.DescribeConflicts(m.links.Apply(ctx, projectPath, changes), assessments), nil
}

func (m ManageProject) inspectProject(ctx context.Context, projectPath string) ([]project.LinkAssessment, error) {
	configured, err := m.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := m.reader.Read(ctx, configured.CatalogPath)
	if err != nil {
		return nil, err
	}
	return m.links.Inspect(ctx, projectPath, configured.EffectiveTargetDirectory(), skills)
}

func (m ManageProject) planChanges(ctx context.Context, assessments []project.LinkAssessment) ([]project.SelectionChange, error) {
	selection, err := m.selection.Choose(ctx, assessments)
	if err != nil {
		return nil, normalizeSelectionError(err)
	}
	return project.Plan(assessments, selection)
}

func (m ManageProject) Run(ctx context.Context, projectPath string) (project.BatchResult, error) {
	return m.Manage(ctx, projectPath)
}

func normalizeSelectionError(err error) error {
	if errors.Is(err, ErrSelectionCanceled) || errors.Is(err, context.Canceled) {
		return ErrSelectionCanceled
	}
	return err
}

type CanceledSelectionUI struct{}

func NewCanceledSelectionUI() CanceledSelectionUI {
	return CanceledSelectionUI{}
}

func (CanceledSelectionUI) Choose(context.Context, []project.LinkAssessment) (project.Selection, error) {
	return project.Selection{}, ErrSelectionCanceled
}
