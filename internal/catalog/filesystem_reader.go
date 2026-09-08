package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FilesystemReader struct{}

func NewFilesystemReader() FilesystemReader {
	return FilesystemReader{}
}

func (FilesystemReader) Read(ctx context.Context, catalogPath string) ([]Skill, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if catalogPath == "" {
		return nil, catalogError(catalogPath, "caminho para um diretório de catálogo", nil)
	}
	absolutePath, err := filepath.Abs(catalogPath)
	if err != nil {
		return nil, catalogError(catalogPath, "diretório absoluto contendo subdiretórios com SKILL.md legível", err)
	}
	entries, err := readCatalogEntries(filepath.Clean(absolutePath))
	if err != nil {
		return nil, err
	}
	skills := collectValidSkills(ctx, filepath.Clean(absolutePath), entries)
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return nil, catalogError(filepath.Clean(absolutePath), "diretório com pelo menos um filho contendo SKILL.md legível", nil)
	}
	sortSkills(skills)
	return skills, nil
}

func readCatalogEntries(catalogPath string) ([]os.DirEntry, error) {
	rootInfo, err := os.Stat(catalogPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, catalogError(catalogPath, "diretório de catálogo existente", err)
	}
	if err != nil {
		return nil, catalogError(catalogPath, "diretório de catálogo legível", err)
	}
	if !rootInfo.IsDir() {
		return nil, catalogError(catalogPath, "diretório de catálogo, não arquivo", nil)
	}
	entries, err := os.ReadDir(catalogPath)
	if err != nil {
		return nil, catalogError(catalogPath, "diretório de catálogo legível", err)
	}
	return entries, nil
}

func collectValidSkills(ctx context.Context, catalogPath string, entries []os.DirEntry) []Skill {
	valid := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if contextError(ctx) != nil {
			return valid
		}
		sourcePath := filepath.Join(catalogPath, entry.Name())
		if isValidSkillDirectory(sourcePath) {
			valid = append(valid, Skill{Name: entry.Name(), SourcePath: filepath.Clean(sourcePath)})
		}
	}
	return valid
}

func isValidSkillDirectory(sourcePath string) bool {
	info, err := os.Stat(sourcePath)
	if err != nil || !info.IsDir() {
		return false
	}
	manifest, err := os.Open(filepath.Join(sourcePath, "SKILL.md"))
	if err != nil {
		return false
	}
	defer manifest.Close()
	manifestInfo, err := manifest.Stat()
	return err == nil && manifestInfo.Mode().IsRegular()
}

func sortSkills(skills []Skill) {
	sort.SliceStable(skills, func(i, j int) bool {
		left, right := foldName(skills[i].Name), foldName(skills[j].Name)
		if left == right {
			return skills[i].Name < skills[j].Name
		}
		return left < right
	})
}

func foldName(name string) string {
	return strings.ToLower(name)
}

func catalogError(value, expected string, cause error) error {
	return &DomainError{Code: ErrorCatalogInvalid, Value: value, Expected: expected, Cause: cause}
}

func contextError(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
