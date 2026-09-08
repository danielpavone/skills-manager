//go:build linux || darwin

package catalog

import (
	"path/filepath"
	"syscall"
	"testing"
)

func TestSkillDiscoveryRejectsFIFOBeforeOpeningManifest(t *testing.T) {
	sourcePath := t.TempDir()
	manifestPath := filepath.Join(sourcePath, "SKILL.md")
	if err := syscall.Mkfifo(manifestPath, 0600); err != nil {
		t.Fatalf("Mkfifo(%q): %v", manifestPath, err)
	}
	if isValidSkillDirectory(sourcePath) {
		t.Fatal("FIFO manifest accepted, want a regular file")
	}
}
