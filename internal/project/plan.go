package project

import (
	"path/filepath"

	"github.com/danielpavone/skills-manager/internal/agentdir"
	"github.com/danielpavone/skills-manager/internal/catalog"
)

type Selection struct {
	SelectedNames []string
}

func NewSelection(selectedNames []string) (Selection, error) {
	selection := Selection{SelectedNames: append([]string(nil), selectedNames...)}
	if err := selection.validateNames(); err != nil {
		return Selection{}, err
	}
	return selection, nil
}

func (s Selection) Contains(skillName string) bool {
	for _, selectedName := range s.SelectedNames {
		if selectedName == skillName {
			return true
		}
	}
	return false
}

func (s Selection) validateNames() error {
	seen := make(map[string]struct{}, len(s.SelectedNames))
	for _, name := range s.SelectedNames {
		if err := validateSkillName(name); err != nil {
			return err
		}
		if _, exists := seen[name]; exists {
			return selectionError(name, "nomes de skills únicos")
		}
		seen[name] = struct{}{}
	}
	return nil
}

type ChangeAction string

const (
	ActionInstall ChangeAction = "install"
	ActionRemove  ChangeAction = "remove"
	ActionKeep    ChangeAction = "keep"
)

type SelectionChange struct {
	Skill    catalog.Skill `json:"skill"`
	LinkPath string        `json:"link_path"`
	Action   ChangeAction  `json:"action"`
}

func Plan(assessments []LinkAssessment, selection Selection) ([]SelectionChange, error) {
	if err := selection.validateNames(); err != nil {
		return nil, err
	}
	knownNames := make(map[string]struct{}, len(assessments))
	changes := make([]SelectionChange, 0, len(assessments))
	for _, assessment := range assessments {
		if err := validateAssessment(assessment); err != nil {
			return nil, err
		}
		if _, exists := knownNames[assessment.Skill.Name]; exists {
			return nil, selectionError(assessment.Skill.Name, "skills do catálogo com nomes únicos")
		}
		knownNames[assessment.Skill.Name] = struct{}{}
		action := actionFor(assessment, selection.Contains(assessment.Skill.Name))
		changes = append(changes, SelectionChange{
			Skill: assessment.Skill, LinkPath: assessment.LinkPath, Action: action,
		})
	}
	for _, name := range selection.SelectedNames {
		if _, exists := knownNames[name]; !exists {
			return nil, selectionError(name, "nome proveniente do catálogo observado")
		}
	}
	return changes, nil
}

func PlanSelection(assessments []LinkAssessment, selection Selection) ([]SelectionChange, error) {
	return Plan(assessments, selection)
}

func actionFor(assessment LinkAssessment, selected bool) ChangeAction {
	if assessment.State == LinkAbsent && selected {
		return ActionInstall
	}
	if assessment.State == LinkInstalled && !selected {
		return ActionRemove
	}
	return ActionKeep
}

func validateAssessment(assessment LinkAssessment) error {
	if err := validateSkill(assessment.Skill); err != nil {
		return err
	}
	if !filepath.IsAbs(assessment.LinkPath) || filepath.Clean(assessment.LinkPath) != assessment.LinkPath {
		return selectionError(assessment.LinkPath, "link_path absoluto e limpo dentro de um diretório de skills suportado")
	}
	if filepath.Base(assessment.LinkPath) != assessment.Skill.Name || !isSkillsDirectory(filepath.Dir(assessment.LinkPath)) {
		return selectionError(assessment.LinkPath, "destino direto <projeto>/{.agents,.claude,.devin}/skills/<nome>")
	}
	if !isKnownLinkState(assessment.State) {
		return selectionError(string(assessment.State), "estado de vínculo válido")
	}
	return nil
}

func validateSkillName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return selectionError(name, "nome de skill não vazio e sem separadores de caminho")
	}
	return nil
}

func isSkillsDirectory(path string) bool {
	_, err := agentdir.FromSkillsPath(path)
	return err == nil
}

func isKnownLinkState(state LinkState) bool {
	switch state {
	case LinkAbsent, LinkInstalled, LinkBroken, LinkWrongTarget, LinkConflict:
		return true
	default:
		return false
	}
}
