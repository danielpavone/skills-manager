package project

import "fmt"

func DescribeConflicts(batch BatchResult, assessments []LinkAssessment) BatchResult {
	conflicts := make(map[string]LinkAssessment)
	for _, assessment := range assessments {
		if assessment.State == LinkBroken || assessment.State == LinkWrongTarget || assessment.State == LinkConflict {
			conflicts[assessment.Skill.Name] = assessment
		}
	}
	for index, result := range batch.Results {
		assessment, exists := conflicts[result.SkillName]
		if exists && result.Outcome == OutcomeUnchanged {
			batch.Results[index].ErrorCode = ErrorLinkConflict
			batch.Results[index].Message = fmt.Sprintf("link_conflict %q: estado %s preservado; esperado destino ausente ou symlink para %q", assessment.LinkPath, assessment.State, assessment.Skill.SourcePath)
		}
	}
	return batch
}
