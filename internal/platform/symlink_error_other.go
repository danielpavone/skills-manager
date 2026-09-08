//go:build !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func IsSymlinkPermissionError(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM)
}

func SymlinkPermissionMessage(skillName, linkPath string, cause error) string {
	return fmt.Sprintf("não foi possível criar o symlink da skill %q em %q: %v; conceda permissão para criar links simbólicos e tente novamente", skillName, linkPath, cause)
}
