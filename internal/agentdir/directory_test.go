package agentdir

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAcceptsSupportedAgentDirectories(t *testing.T) {
	for _, value := range []string{".agents", ".claude", ".devin"} {
		directory, err := Parse(value)
		if err != nil || string(directory) != value {
			t.Fatalf("Parse(%q) = %q, %v", value, directory, err)
		}
	}
}

func TestParseRejectsUnsupportedAgentDirectory(t *testing.T) {
	_, err := Parse(".cursor")
	if err == nil || !strings.Contains(err.Error(), ".cursor") {
		t.Fatalf("Parse() error = %v, want offending directory", err)
	}
}

func TestSkillsPathUsesSelectedDirectory(t *testing.T) {
	root := filepath.Join("project", "root")
	want := filepath.Join(root, ".claude", "skills")
	if got := SkillsPath(root, Claude); got != want {
		t.Fatalf("SkillsPath() = %q, want %q", got, want)
	}
}

func TestFromSkillsPathRecognizesSupportedDirectory(t *testing.T) {
	directory, err := FromSkillsPath(filepath.Join("project", ".devin", "skills"))
	if err != nil || directory != Devin {
		t.Fatalf("FromSkillsPath() = %q, %v, want .devin", directory, err)
	}
}
