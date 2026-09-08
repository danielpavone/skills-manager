package project

import (
	"errors"
	"os"
	"path/filepath"
)

func validateLocalParents(fileSystem FileSystem, linkPath string) error {
	skillsPath := filepath.Dir(linkPath)
	for _, parent := range []string{filepath.Dir(skillsPath), skillsPath} {
		info, err := fileSystem.Lstat(parent)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return filesystemError(parent, "diretório local inspecionável", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return &DomainError{Code: ErrorLinkConflict, Value: parent, Expected: "diretório real do projeto, sem redirecionamento por symlink"}
		}
	}
	return nil
}
