//go:build windows

package platform

import (
	"errors"
	"fmt"
	"os"
)

func IsSymlinkPermissionError(err error) bool {
	return errors.Is(err, os.ErrPermission)
}

func SymlinkPermissionMessage(skillName, linkPath string, cause error) string {
	return fmt.Sprintf("não foi possível criar o symlink da skill %q em %q: %v; habilite o Developer Mode ou execute com privilégio adequado", skillName, linkPath, cause)
}
