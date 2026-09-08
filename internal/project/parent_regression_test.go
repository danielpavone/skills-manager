package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyRejectsSymlinkedSkillsParentWithoutRemovingExternalLink(t *testing.T) {
	projectRoot, outside := t.TempDir(), t.TempDir()
	skill := createApplySkill(t, t.TempDir(), "protected")
	if err := os.Mkdir(filepath.Join(projectRoot, ".agents"), 0700); err != nil {
		t.Fatal(err)
	}
	makeSymlink(t, outside, filepath.Join(projectRoot, ".agents", "skills"))
	outsideLink := filepath.Join(outside, skill.name)
	makeSymlink(t, skill.sourcePath, outsideLink)
	change := installChanges(projectRoot, skill)[0]
	change.Action = ActionRemove
	result := NewFilesystemLinks().Apply(context.Background(), projectRoot, []SelectionChange{change})
	if !result.HasFailures {
		t.Fatalf("result = %#v, want rejection of symlinked parent", result)
	}
	if _, err := os.Readlink(outsideLink); err != nil {
		t.Fatalf("external link %q was changed: %v", outsideLink, err)
	}
}

func TestApplyRejectsSymlinkedAgentsParentWithoutCreatingExternalDirectory(t *testing.T) {
	projectRoot, outside := t.TempDir(), t.TempDir()
	skill := createApplySkill(t, t.TempDir(), "protected")
	makeSymlink(t, outside, filepath.Join(projectRoot, ".agents"))
	result := NewFilesystemLinks().Apply(context.Background(), projectRoot, installChanges(projectRoot, skill))
	if !result.HasFailures {
		t.Fatalf("result = %#v, want rejection of symlinked parent", result)
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("external entries = %v, %v, want empty directory", entries, err)
	}
}
