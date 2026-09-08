package project

import (
	"strings"
	"testing"
)

func TestDescribeConflictsRetainsLockedStatesInUnchangedSummary(t *testing.T) {
	assessments := planAssessments(t, LinkBroken, LinkWrongTarget, LinkConflict, LinkAbsent)
	changes, err := Plan(assessments, Selection{})
	if err != nil {
		t.Fatal(err)
	}
	batch := DescribeConflicts(UnchangedBatch(changes), assessments)
	for index, result := range batch.Results[:3] {
		if result.ErrorCode != ErrorLinkConflict || !strings.Contains(result.Message, string(assessments[index].State)) {
			t.Fatalf("result = %#v, want preserved conflict details", result)
		}
	}
	if batch.HasFailures || batch.Results[3].Message != "" {
		t.Fatalf("batch = %#v, want unchanged without operation failures", batch)
	}
}
