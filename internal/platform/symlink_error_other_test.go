//go:build !windows

package platform

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestSymlinkPermissionErrorRecognizesPortablePermissionFailures(t *testing.T) {
	for _, cause := range []error{os.ErrPermission, syscall.EACCES, syscall.EPERM} {
		if !IsSymlinkPermissionError(cause) {
			t.Errorf("IsSymlinkPermissionError(%v) = false, want true", cause)
		}
	}
	if IsSymlinkPermissionError(errors.New("unrelated filesystem failure")) {
		t.Fatal("IsSymlinkPermissionError(unrelated) = true, want false")
	}
}

func TestSymlinkPermissionMessageContainsOffendingSkillAndPosixGuidance(t *testing.T) {
	message := SymlinkPermissionMessage("code-review", "/project/.agents/skills/code-review", os.ErrPermission)
	for _, expected := range []string{"code-review", "/project/.agents/skills/code-review", "permissão", "links simbólicos"} {
		if !strings.Contains(message, expected) {
			t.Errorf("message = %q, want %q", message, expected)
		}
	}
}
