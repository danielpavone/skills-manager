package project

import "time"

type OperationOutcome string

const (
	OutcomeInstalled OperationOutcome = "installed"
	OutcomeRemoved   OperationOutcome = "removed"
	OutcomeUnchanged OperationOutcome = "unchanged"
	OutcomeFailed    OperationOutcome = "failed"
)

type OperationResult struct {
	SkillName string           `json:"skill_name"`
	Action    ChangeAction     `json:"action"`
	Outcome   OperationOutcome `json:"outcome"`
	ErrorCode ErrorCode        `json:"error_code,omitempty"`
	Message   string           `json:"message,omitempty"`
}

type BatchResult struct {
	Results     []OperationResult `json:"results"`
	HasFailures bool              `json:"has_failures"`
	Elapsed     time.Duration     `json:"elapsed"`
}

func OnlyKeeps(changes []SelectionChange) bool {
	for _, change := range changes {
		if change.Action != ActionKeep {
			return false
		}
	}
	return true
}

func UnchangedBatch(changes []SelectionChange) BatchResult {
	results := make([]OperationResult, 0, len(changes))
	for _, change := range changes {
		results = append(results, OperationResult{
			SkillName: change.Skill.Name,
			Action:    change.Action,
			Outcome:   OutcomeUnchanged,
		})
	}
	return BatchResult{Results: results}
}
