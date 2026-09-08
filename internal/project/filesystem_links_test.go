package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/danielpavone/skills-manager/internal/agentdir"
	"github.com/danielpavone/skills-manager/internal/catalog"
)

func TestFilesystemLinksApplyInstallsAbsoluteLinksInEmptyProject(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skills := createApplySkills(t, catalogRoot, "first", "second")
	changes := installChanges(projectRoot, skills...)

	result := NewFilesystemLinks().Apply(context.Background(), projectRoot, changes)
	if result.HasFailures || !reflect.DeepEqual(operationOutcomes(result), []OperationOutcome{OutcomeInstalled, OutcomeInstalled}) {
		t.Fatalf("result = %#v, want two successful installations", result)
	}
	for _, skill := range skills {
		linkPath := filepath.Join(projectRoot, ".agents", "skills", skill.name)
		actual, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Readlink(%q) error = %v", linkPath, err)
		}
		if actual != skill.sourcePath || !filepath.IsAbs(actual) {
			t.Fatalf("link target = %q, want absolute %q", actual, skill.sourcePath)
		}
	}
}

func TestFilesystemLinksApplyInstallsInClaudeAndDevinDirectories(t *testing.T) {
	for _, target := range []agentdir.Directory{agentdir.Claude, agentdir.Devin} {
		t.Run(string(target), func(t *testing.T) {
			catalogRoot := t.TempDir()
			projectRoot := t.TempDir()
			skill := createApplySkill(t, catalogRoot, "tdd")
			linkPath := filepath.Join(agentdir.SkillsPath(projectRoot, target), skill.name)
			change := SelectionChange{Skill: catalogSkill(skill), LinkPath: linkPath, Action: ActionInstall}

			result := NewFilesystemLinks().Apply(context.Background(), projectRoot, []SelectionChange{change})
			actual, err := os.Readlink(linkPath)
			if result.HasFailures || err != nil || actual != skill.sourcePath {
				t.Fatalf("result = %#v, link = %q, error = %v; want %q", result, actual, err, skill.sourcePath)
			}
		})
	}
}

func TestFilesystemLinksApplyRemovesOnlyTheLocalCorrectLink(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skill := createApplySkill(t, catalogRoot, "local-only")
	change := installChanges(projectRoot, skill)[0]
	links := NewFilesystemLinks()
	if result := links.Apply(context.Background(), projectRoot, []SelectionChange{change}); result.HasFailures {
		t.Fatalf("initial installation failed: %#v", result)
	}

	change.Action = ActionRemove
	result := links.Apply(context.Background(), projectRoot, []SelectionChange{change})
	if result.HasFailures || result.Results[0].Outcome != OutcomeRemoved {
		t.Fatalf("removal result = %#v, want removed", result)
	}
	if _, err := os.Lstat(change.LinkPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("local link = %v, want absent", err)
	}
	content, err := os.ReadFile(filepath.Join(skill.sourcePath, "SKILL.md"))
	if err != nil || string(content) != "# local-only" {
		t.Fatalf("source content = %q, error = %v; want unchanged source", content, err)
	}
}

func TestFilesystemLinksApplyReturnsStateChangedWithoutOverwritingRaceConflict(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skill := createApplySkill(t, catalogRoot, "race")
	change := installChanges(projectRoot, skill)[0]
	assessment, err := InspectLink(context.Background(), projectRoot, catalogSkill(skill))
	if err != nil || assessment.State != LinkAbsent {
		t.Fatalf("initial assessment = %#v, error = %v; want absent", assessment, err)
	}
	if err := os.MkdirAll(filepath.Dir(change.LinkPath), 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(change.LinkPath, []byte("protected"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result := NewFilesystemLinks().Apply(context.Background(), projectRoot, []SelectionChange{change})
	if result.Results[0].ErrorCode != ErrorStateChanged {
		t.Fatalf("error code = %q, want state_changed", result.Results[0].ErrorCode)
	}
	content, err := os.ReadFile(change.LinkPath)
	if err != nil || string(content) != "protected" {
		t.Fatalf("conflict content = %q, error = %v; want preserved file", content, err)
	}
}

func TestFilesystemLinksApplyContinuesAfterIndividualFailureInCatalogOrder(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skills := createApplySkills(t, catalogRoot, "first", "second", "third")
	changes := installChanges(projectRoot, skills...)
	failingPath := changes[1].LinkPath
	fake := &failingSymlinkFileSystem{FileSystem: osFileSystem{}, failingPath: failingPath, failure: os.ErrPermission}

	result := NewFilesystemLinksWithFileSystem(fake).Apply(context.Background(), projectRoot, changes)
	if !result.HasFailures {
		t.Fatal("HasFailures = false, want true")
	}
	if got := operationNames(result); !reflect.DeepEqual(got, []string{"first", "second", "third"}) {
		t.Fatalf("result names = %#v, want catalog order", got)
	}
	if got := operationOutcomes(result); !reflect.DeepEqual(got, []OperationOutcome{OutcomeInstalled, OutcomeFailed, OutcomeInstalled}) {
		t.Fatalf("result outcomes = %#v, want installed/failed/installed", got)
	}
	if result.Results[1].ErrorCode != ErrorSymlinkPermissionDenied {
		t.Fatalf("permission code = %q, want symlink_permission_denied", result.Results[1].ErrorCode)
	}
	if _, err := os.Lstat(changes[2].LinkPath); err != nil {
		t.Fatalf("third link error = %v, want later item processed", err)
	}
}

func TestFilesystemLinksApplyKeepsAlreadyCorrectLinksWithoutMutation(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skill := createApplySkill(t, catalogRoot, "stable")
	change := installChanges(projectRoot, skill)[0]
	links := NewFilesystemLinks()
	if result := links.Apply(context.Background(), projectRoot, []SelectionChange{change}); result.HasFailures {
		t.Fatalf("initial installation failed: %#v", result)
	}
	change.Action = ActionKeep
	result := links.Apply(context.Background(), projectRoot, []SelectionChange{change})
	if result.HasFailures || result.Results[0].Outcome != OutcomeUnchanged {
		t.Fatalf("keep result = %#v, want unchanged", result)
	}
}

func TestFilesystemLinksApplyPlanKeepsLinksWhenProjectReopens(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skill := createApplySkill(t, catalogRoot, "reopened")
	links := NewFilesystemLinks()
	change := installChanges(projectRoot, skill)[0]
	if result := links.Apply(context.Background(), projectRoot, []SelectionChange{change}); result.HasFailures {
		t.Fatalf("initial installation failed: %#v", result)
	}
	assessments, err := links.Inspect(context.Background(), projectRoot, agentdir.Agents, []catalog.Skill{catalogSkill(skill)})
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	selection, err := NewSelection([]string{"reopened"})
	if err != nil {
		t.Fatalf("NewSelection() error = %v", err)
	}
	planned, err := Plan(assessments, selection)
	if err != nil || planned[0].Action != ActionKeep {
		t.Fatalf("planned = %#v, error = %v; want keep", planned, err)
	}
	fake := &countingSymlinkFileSystem{FileSystem: osFileSystem{}}
	result := NewFilesystemLinksWithFileSystem(fake).Apply(context.Background(), projectRoot, planned)
	if result.HasFailures || fake.symlinkCalls != 0 {
		t.Fatalf("reopen result = %#v, symlink calls = %d; want unchanged without recreation", result, fake.symlinkCalls)
	}
}

func TestFilesystemLinksApplyPropagatesCentralSourceChangesToTwoProjects(t *testing.T) {
	catalogRoot := t.TempDir()
	firstProject, secondProject := t.TempDir(), t.TempDir()
	skill := createApplySkill(t, catalogRoot, "shared")
	links := NewFilesystemLinks()
	for _, projectRoot := range []string{firstProject, secondProject} {
		if result := links.Apply(context.Background(), projectRoot, installChanges(projectRoot, skill)); result.HasFailures {
			t.Fatalf("installation in %q failed: %#v", projectRoot, result)
		}
	}
	sourceManifest := filepath.Join(skill.sourcePath, "SKILL.md")
	if err := os.WriteFile(sourceManifest, []byte("# changed centrally"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	for _, projectRoot := range []string{firstProject, secondProject} {
		linkManifest := filepath.Join(projectRoot, ".agents", "skills", "shared", "SKILL.md")
		content, err := os.ReadFile(linkManifest)
		if err != nil || string(content) != "# changed centrally" {
			t.Fatalf("project %q content = %q, error = %v; want central update", projectRoot, content, err)
		}
	}
}

func TestFilesystemLinksApplyTenSkillsWithinLocalBudget(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	skills := make([]skillForTest, 0, 10)
	for index := 0; index < 10; index++ {
		skills = append(skills, createApplySkill(t, catalogRoot, "skill-"+string(rune('a'+index))))
	}
	started := time.Now()
	result := NewFilesystemLinks().Apply(context.Background(), projectRoot, installChanges(projectRoot, skills...))
	if result.HasFailures {
		t.Fatalf("batch failed: %#v", result)
	}
	if result.Elapsed > 5*time.Second || time.Since(started) > 5*time.Second {
		t.Fatalf("elapsed = %v, want less than five seconds", result.Elapsed)
	}
}

type skillForTest struct {
	name       string
	sourcePath string
}

type failingSymlinkFileSystem struct {
	FileSystem
	failingPath string
	failure     error
}

type countingSymlinkFileSystem struct {
	FileSystem
	symlinkCalls int
}

func (f *countingSymlinkFileSystem) Symlink(source, destination string) error {
	f.symlinkCalls++
	return f.FileSystem.Symlink(source, destination)
}

func (f *failingSymlinkFileSystem) Symlink(source, destination string) error {
	if destination == f.failingPath {
		return f.failure
	}
	return f.FileSystem.Symlink(source, destination)
}

func createApplySkills(t *testing.T, root string, names ...string) []skillForTest {
	created := make([]skillForTest, 0, len(names))
	for _, name := range names {
		created = append(created, createApplySkill(t, root, name))
	}
	return created
}

func createApplySkill(t *testing.T, root, name string) skillForTest {
	t.Helper()
	sourcePath := filepath.Join(root, name)
	if err := os.MkdirAll(sourcePath, 0700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", sourcePath, err)
	}
	if err := os.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("# "+name), 0600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", name, err)
	}
	return skillForTest{name: name, sourcePath: sourcePath}
}

func catalogSkill(skill skillForTest) catalog.Skill {
	return catalog.Skill{Name: skill.name, SourcePath: skill.sourcePath}
}

func installChanges(projectRoot string, skills ...skillForTest) []SelectionChange {
	changes := make([]SelectionChange, 0, len(skills))
	for _, skill := range skills {
		changes = append(changes, SelectionChange{
			Skill: catalogSkill(skill), LinkPath: filepath.Join(projectRoot, ".agents", "skills", skill.name), Action: ActionInstall,
		})
	}
	return changes
}

func operationNames(result BatchResult) []string {
	names := make([]string, 0, len(result.Results))
	for _, operation := range result.Results {
		names = append(names, operation.SkillName)
	}
	return names
}

func operationOutcomes(result BatchResult) []OperationOutcome {
	outcomes := make([]OperationOutcome, 0, len(result.Results))
	for _, operation := range result.Results {
		outcomes = append(outcomes, operation.Outcome)
	}
	return outcomes
}
