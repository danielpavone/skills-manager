//go:build windows

package platform

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestSymlinkPermissionErrorRecognizesWindowsPrivilegeFailure(t *testing.T) {
	cause := &os.LinkError{Op: "symlink", Old: `C:\catalog\skill`, New: `C:\project\skill`, Err: syscall.ERROR_PRIVILEGE_NOT_HELD}
	if !IsSymlinkPermissionError(cause) || !IsSymlinkPermissionError(os.ErrPermission) {
		t.Fatal("permission detection = false, want true for denied privilege and access")
	}
	if IsSymlinkPermissionError(errors.New("unrelated failure")) {
		t.Fatal("permission detection = true, want false for unrelated failure")
	}
}

func TestSymlinkPermissionMessageContainsWindowsGuidance(t *testing.T) {
	message := SymlinkPermissionMessage("code-review", `C:\project\.agents\skills\code-review`, os.ErrPermission)
	for _, expected := range []string{"code-review", "Developer Mode", "privilégio"} {
		if !strings.Contains(message, expected) {
			t.Errorf("message = %q, want %q", message, expected)
		}
	}
}
