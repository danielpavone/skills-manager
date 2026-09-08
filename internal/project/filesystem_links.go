package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/danielpavone/skills-manager/internal/platform"
)

func (f FilesystemLinks) Apply(ctx context.Context, projectPath string, changes []SelectionChange) BatchResult {
	started := time.Now()
	results := make([]OperationResult, 0, len(changes))
	for _, change := range changes {
		result := f.applyChange(ctx, projectPath, change)
		results = append(results, result)
	}
	return BatchResult{Results: results, HasFailures: hasFailures(results), Elapsed: time.Since(started)}
}

func (f FilesystemLinks) applyChange(ctx context.Context, projectPath string, change SelectionChange) OperationResult {
	base := OperationResult{SkillName: change.Skill.Name, Action: change.Action}
	if err := validateChange(projectPath, change); err != nil {
		return failedResult(base, err)
	}
	if change.Action == ActionKeep {
		base.Outcome = OutcomeUnchanged
		return base
	}
	if err := contextError(ctx); err != nil {
		return failedResult(base, filesystemError(change.Skill.Name, "contexto ativo antes da mutação", err))
	}
	if change.Action == ActionInstall {
		return f.installLink(ctx, projectPath, change, base)
	}
	if change.Action == ActionRemove {
		return f.removeLink(ctx, projectPath, change, base)
	}
	return failedResult(base, selectionError(string(change.Action), "ação install, remove ou keep"))
}

func validateChange(projectPath string, change SelectionChange) error {
	if err := validateSkill(change.Skill); err != nil {
		return err
	}
	absoluteProject, err := absoluteProjectPath(projectPath)
	if err != nil {
		return err
	}
	expectedPath := filepath.Join(absoluteProject, ".agents", "skills", change.Skill.Name)
	if change.LinkPath != expectedPath {
		return selectionError(change.LinkPath, "destino direto do projeto atual em .agents/skills")
	}
	if change.Action != ActionInstall && change.Action != ActionRemove && change.Action != ActionKeep {
		return selectionError(string(change.Action), "ação install, remove ou keep")
	}
	return nil
}

func (f FilesystemLinks) installLink(ctx context.Context, projectPath string, change SelectionChange, base OperationResult) OperationResult {
	fileSystem := f.withDefaultFileSystem().fileSystem
	parent := filepath.Dir(change.LinkPath)
	if err := validateLocalParents(fileSystem, change.LinkPath); err != nil {
		return failedResult(base, err)
	}
	if err := fileSystem.MkdirAll(parent, 0700); err != nil {
		return failedResult(base, filesystemError(parent, "diretório .agents/skills criável", err))
	}
	assessment, err := inspectSkillLink(ctx, fileSystem, projectPath, change.Skill)
	if err != nil {
		return failedResult(base, err)
	}
	if assessment.State != LinkAbsent || assessment.LinkPath != change.LinkPath {
		return failedResult(base, stateChangedError(change.LinkPath, "destino ausente imediatamente antes da criação"))
	}
	return createAbsoluteSymlink(fileSystem, change, base)
}

func createAbsoluteSymlink(fileSystem FileSystem, change SelectionChange, base OperationResult) OperationResult {
	if err := fileSystem.Symlink(change.Skill.SourcePath, change.LinkPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return failedResult(base, stateChangedError(change.LinkPath, "destino ausente no instante da criação"))
		}
		return failedResult(base, translateSymlinkError(change.Skill.Name, change.LinkPath, err))
	}
	base.Outcome = OutcomeInstalled
	return base
}

func (f FilesystemLinks) removeLink(ctx context.Context, projectPath string, change SelectionChange, base OperationResult) OperationResult {
	fileSystem := f.withDefaultFileSystem().fileSystem
	assessment, err := inspectSkillLink(ctx, fileSystem, projectPath, change.Skill)
	if err != nil {
		return failedResult(base, err)
	}
	if assessment.State != LinkInstalled || assessment.LinkPath != change.LinkPath {
		return failedResult(base, stateChangedError(change.LinkPath, "symlink apontando para a origem esperada imediatamente antes da remoção"))
	}
	if err := fileSystem.Remove(change.LinkPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return failedResult(base, stateChangedError(change.LinkPath, "symlink local existente no instante da remoção"))
		}
		return failedResult(base, filesystemError(change.LinkPath, "somente o symlink local removível", err))
	}
	base.Outcome = OutcomeRemoved
	return base
}

func translateSymlinkError(skillName, linkPath string, cause error) error {
	if platform.IsSymlinkPermissionError(cause) {
		message := platform.SymlinkPermissionMessage(skillName, linkPath, cause)
		return &DomainError{Code: ErrorSymlinkPermissionDenied, Value: skillName, Expected: "symlink criável no destino local", Cause: errors.New(message)}
	}
	return filesystemError(linkPath, "symlink absoluto criável para a origem da skill", cause)
}

func stateChangedError(value, expected string) error {
	return &DomainError{Code: ErrorStateChanged, Value: value, Expected: expected}
}

func failedResult(base OperationResult, err error) OperationResult {
	base.Outcome = OutcomeFailed
	base.Message = err.Error()
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		base.ErrorCode = domainErr.Code
	} else {
		base.ErrorCode = ErrorFilesystemFailure
	}
	return base
}

func hasFailures(results []OperationResult) bool {
	for _, result := range results {
		if result.Outcome == OutcomeFailed {
			return true
		}
	}
	return false
}
