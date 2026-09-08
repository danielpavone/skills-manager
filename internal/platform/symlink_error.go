package platform

import (
	"errors"
	"fmt"
	"os"
)

func IsSymlinkPermissionError(err error) bool {
	permissionDenied := errors.Is(err, os.ErrPermission)
	return permissionDenied
}

func SymlinkPermissionMessage(skillName, linkPath string, cause error) string {
	message := fmt.Sprintf("não foi possível criar o symlink da skill %q em %q: %v; conceda permissão para criar links simbólicos e tente novamente", skillName, linkPath, cause)
	return message
}
