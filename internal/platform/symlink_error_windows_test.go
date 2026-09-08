//go:build windows

package platform

import (
	"os"
	"strings"
	"testing"
)

func TestSymlinkPermissionMessageContainsWindowsGuidance(t *testing.T) {
	message := SymlinkPermissionMessage("code-review", `C:\project\.agents\skills\code-review`, os.ErrPermission)
	for _, expected := range []string{"code-review", "Developer Mode", "privilégio"} {
		if !strings.Contains(message, expected) {
			t.Errorf("message = %q, want %q", message, expected)
		}
	}
}
