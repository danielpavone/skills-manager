package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/danielpavone/skills-manager/internal/agentdir"
	"github.com/danielpavone/skills-manager/internal/catalog"
)

func TestFilesystemLinksClassifiesFiveLinkStates(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	otherRoot := t.TempDir()
	makeProjectSkillsDirectory(t, projectRoot)
	makeCatalogSkills(t, catalogRoot, "absent", "installed", "broken", "wrong", "conflict")
	makeSymlink(t, filepath.Join(catalogRoot, "installed"), filepath.Join(projectRoot, ".agents/skills/installed"))
	makeSymlink(t, filepath.Join(catalogRoot, "missing"), filepath.Join(projectRoot, ".agents/skills/broken"))
	makeSymlink(t, filepath.Join(otherRoot), filepath.Join(projectRoot, ".agents/skills/wrong"))
	writeFile(t, filepath.Join(projectRoot, ".agents/skills/conflict"))

	skills := catalogSkills(catalogRoot, "absent", "installed", "broken", "wrong", "conflict")
	assessments, err := NewFilesystemLinks().Inspect(context.Background(), projectRoot, agentdir.Agents, skills)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	want := []LinkState{LinkAbsent, LinkInstalled, LinkBroken, LinkWrongTarget, LinkConflict}
	got := assessmentStates(assessments)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("states = %#v, want %#v", got, want)
	}
	if assessments[2].ActualTarget == nil || *assessments[2].ActualTarget != filepath.Join(catalogRoot, "missing") {
		t.Fatalf("broken target = %#v, want textual symlink target", assessments[2].ActualTarget)
	}
}

func TestInspectLinkUsesAbsoluteCleanDestinationAndSameFileIdentity(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	makeProjectSkillsDirectory(t, projectRoot)
	makeCatalogSkills(t, catalogRoot, "identity")
	linkPath := filepath.Join(projectRoot, ".agents/skills/identity")
	makeSymlink(t, filepath.Join(catalogRoot, "identity"), linkPath)

	assessment, err := InspectLink(context.Background(), projectRoot, catalog.Skill{
		Name: "identity", SourcePath: filepath.Join(catalogRoot, "identity"),
	})
	if err != nil {
		t.Fatalf("InspectLink() error = %v", err)
	}
	if assessment.State != LinkInstalled {
		t.Fatalf("state = %q, want installed", assessment.State)
	}
	if !filepath.IsAbs(assessment.LinkPath) || filepath.Clean(assessment.LinkPath) != assessment.LinkPath {
		t.Fatalf("LinkPath = %q, want absolute clean path", assessment.LinkPath)
	}
}

func TestFilesystemLinksRejectsCanceledInspection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewFilesystemLinks().Inspect(ctx, t.TempDir(), agentdir.Agents, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect() error = %v, want context canceled", err)
	}
}

func TestFilesystemLinksInspectsConfiguredDevinDirectory(t *testing.T) {
	catalogRoot := t.TempDir()
	projectRoot := t.TempDir()
	makeCatalogSkills(t, catalogRoot, "tdd")

	assessments, err := NewFilesystemLinks().Inspect(
		context.Background(), projectRoot, agentdir.Devin, catalogSkills(catalogRoot, "tdd"),
	)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	want := filepath.Join(projectRoot, ".devin", "skills", "tdd")
	if assessments[0].LinkPath != want || assessments[0].State != LinkAbsent {
		t.Fatalf("assessment = %#v, want absent link at %q", assessments[0], want)
	}
}

func makeCatalogSkills(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, name := range names {
		path := filepath.Join(root, name)
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", name, err)
		}
		writeFile(t, filepath.Join(path, "SKILL.md"))
	}
}

func catalogSkills(root string, names ...string) []catalog.Skill {
	skills := make([]catalog.Skill, 0, len(names))
	for _, name := range names {
		skills = append(skills, catalog.Skill{Name: name, SourcePath: filepath.Join(root, name)})
	}
	return skills
}

func makeProjectSkillsDirectory(t *testing.T, projectRoot string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".agents/skills"), 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
}

func makeSymlink(t *testing.T, source, target string) {
	t.Helper()
	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("Symlink(%q, %q) error = %v", source, target, err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("content"), 0600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assessmentStates(assessments []LinkAssessment) []LinkState {
	states := make([]LinkState, 0, len(assessments))
	for _, assessment := range assessments {
		states = append(states, assessment.State)
	}
	return states
}
