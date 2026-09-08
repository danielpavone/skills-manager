package agentdir

import (
	"fmt"
	"path/filepath"
)

type Directory string

const (
	Agents Directory = ".agents"
	Claude Directory = ".claude"
	Devin  Directory = ".devin"
)

func Parse(value string) (Directory, error) {
	directory := Directory(value)
	if directory == Agents || directory == Claude || directory == Devin {
		return directory, nil
	}
	return "", fmt.Errorf("diretório de agente %q: esperado .agents, .claude ou .devin", value)
}

func SkillsPath(root string, directory Directory) string {
	return filepath.Join(root, string(directory), "skills")
}

func FromSkillsPath(path string) (Directory, error) {
	if filepath.Base(path) != "skills" {
		return "", fmt.Errorf("caminho de skills %q: esperado <raiz>/.agents/skills, <raiz>/.claude/skills ou <raiz>/.devin/skills", path)
	}
	return Parse(filepath.Base(filepath.Dir(path)))
}
