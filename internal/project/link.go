package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielpavone/skills-manager/internal/catalog"
)

type LinkState string

const (
	LinkAbsent      LinkState = "absent"
	LinkInstalled   LinkState = "installed"
	LinkBroken      LinkState = "broken"
	LinkWrongTarget LinkState = "wrong_target"
	LinkConflict    LinkState = "conflict"
)

type ErrorCode string

const (
	ErrorSelectionInvalid        ErrorCode = "selection_invalid"
	ErrorLinkConflict            ErrorCode = "link_conflict"
	ErrorStateChanged            ErrorCode = "state_changed"
	ErrorSymlinkPermissionDenied ErrorCode = "symlink_permission_denied"
	ErrorFilesystemFailure       ErrorCode = "filesystem_failure"
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

type LinkAssessment struct {
	Skill        catalog.Skill `json:"skill"`
	LinkPath     string        `json:"link_path"`
	State        LinkState     `json:"state"`
	ActualTarget *string       `json:"actual_target"`
}

type FileSystem interface {
	Lstat(name string) (os.FileInfo, error)
	Readlink(name string) (string, error)
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, permission os.FileMode) error
	Symlink(source, destination string) error
	Remove(path string) error
}

type osFileSystem struct{}

func (osFileSystem) Lstat(name string) (os.FileInfo, error) { return os.Lstat(name) }

func (osFileSystem) Readlink(name string) (string, error) { return os.Readlink(name) }

func (osFileSystem) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

func (osFileSystem) MkdirAll(path string, permission os.FileMode) error {
	return os.MkdirAll(path, permission)
}

func (osFileSystem) Symlink(source, destination string) error {
	return os.Symlink(source, destination)
}

func (osFileSystem) Remove(path string) error { return os.Remove(path) }

type FilesystemLinks struct {
	fileSystem FileSystem
}

func NewFilesystemLinks() FilesystemLinks {
	return FilesystemLinks{fileSystem: osFileSystem{}}
}

func NewFilesystemLinksWithFileSystem(fileSystem FileSystem) FilesystemLinks {
	return FilesystemLinks{fileSystem: fileSystem}
}

func (f FilesystemLinks) Inspect(ctx context.Context, projectPath string, skills []catalog.Skill) ([]LinkAssessment, error) {
	f = f.withDefaultFileSystem()
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	absoluteProject, err := absoluteProjectPath(projectPath)
	if err != nil {
		return nil, err
	}
	assessments := make([]LinkAssessment, 0, len(skills))
	for _, skill := range skills {
		if err := contextError(ctx); err != nil {
			return nil, err
		}
		assessment, err := inspectSkillLink(ctx, f.fileSystem, absoluteProject, skill)
		if err != nil {
			return nil, err
		}
		assessments = append(assessments, assessment)
	}
	return assessments, nil
}

func InspectLink(ctx context.Context, projectPath string, skill catalog.Skill) (LinkAssessment, error) {
	return NewFilesystemLinks().inspectOne(ctx, projectPath, skill)
}

func (f FilesystemLinks) inspectOne(ctx context.Context, projectPath string, skill catalog.Skill) (LinkAssessment, error) {
	f = f.withDefaultFileSystem()
	if err := contextError(ctx); err != nil {
		return LinkAssessment{}, err
	}
	absoluteProject, err := absoluteProjectPath(projectPath)
	if err != nil {
		return LinkAssessment{}, err
	}
	return inspectSkillLink(ctx, f.fileSystem, absoluteProject, skill)
}

func (f FilesystemLinks) withDefaultFileSystem() FilesystemLinks {
	if f.fileSystem != nil {
		return f
	}
	return NewFilesystemLinks()
}

func inspectSkillLink(ctx context.Context, fileSystem FileSystem, projectPath string, skill catalog.Skill) (LinkAssessment, error) {
	if err := validateSkill(skill); err != nil {
		return LinkAssessment{}, err
	}
	linkPath := filepath.Join(projectPath, ".agents", "skills", skill.Name)
	if err := contextError(ctx); err != nil {
		return LinkAssessment{}, err
	}
	if _, err := fileSystem.Stat(skill.SourcePath); err != nil {
		return LinkAssessment{}, filesystemError(skill.SourcePath, "origem da skill existente e acessível", err)
	}
	linkInfo, err := fileSystem.Lstat(linkPath)
	if errors.Is(err, os.ErrNotExist) {
		return LinkAssessment{Skill: skill, LinkPath: linkPath, State: LinkAbsent}, nil
	}
	if err != nil {
		return LinkAssessment{}, filesystemError(linkPath, "destino local inspecionável com Lstat", err)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return LinkAssessment{Skill: skill, LinkPath: linkPath, State: LinkConflict}, nil
	}
	return classifySymlink(fileSystem, linkPath, skill, linkInfo)
}

func classifySymlink(fileSystem FileSystem, linkPath string, skill catalog.Skill, _ os.FileInfo) (LinkAssessment, error) {
	actual, err := fileSystem.Readlink(linkPath)
	if err != nil {
		return LinkAssessment{}, filesystemError(linkPath, "link simbólico legível", err)
	}
	actualTarget := actual
	assessment := LinkAssessment{
		Skill: skill, LinkPath: linkPath, ActualTarget: &actualTarget,
	}
	resolved, err := fileSystem.Stat(linkPath)
	if errors.Is(err, os.ErrNotExist) {
		assessment.State = LinkBroken
		return assessment, nil
	}
	if err != nil {
		return LinkAssessment{}, filesystemError(linkPath, "destino do link simbólico acessível", err)
	}
	sourceInfo, err := fileSystem.Stat(skill.SourcePath)
	if err != nil {
		return LinkAssessment{}, filesystemError(skill.SourcePath, "origem da skill existente e acessível", err)
	}
	assessment.State = LinkWrongTarget
	if os.SameFile(sourceInfo, resolved) {
		assessment.State = LinkInstalled
	}
	return assessment, nil
}

func validateSkill(skill catalog.Skill) error {
	if skill.Name == "" || skill.Name == "." || skill.Name == ".." || filepath.Base(skill.Name) != skill.Name {
		return selectionError(skill.Name, "nome de skill não vazio e sem separadores de caminho")
	}
	if !filepath.IsAbs(skill.SourcePath) || filepath.Clean(skill.SourcePath) != skill.SourcePath {
		return selectionError(skill.SourcePath, "source_path absoluto e limpo")
	}
	return nil
}

func absoluteProjectPath(projectPath string) (string, error) {
	if projectPath == "" {
		return "", filesystemError(projectPath, "caminho do projeto não vazio", nil)
	}
	absolute, err := filepath.Abs(projectPath)
	if err != nil {
		return "", filesystemError(projectPath, "caminho absoluto do projeto", err)
	}
	return filepath.Clean(absolute), nil
}

func selectionError(value, expected string) error {
	return &DomainError{Code: ErrorSelectionInvalid, Value: value, Expected: expected}
}

func filesystemError(value, expected string, cause error) error {
	return &DomainError{Code: ErrorFilesystemFailure, Value: value, Expected: expected, Cause: cause}
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
