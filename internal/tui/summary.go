package tui

import (
	"github.com/danielpavone/skills-manager/internal/project"
	"github.com/danielpavone/skills-manager/internal/summary"
)

func RenderSummary(result project.BatchResult) string {
	return summary.RenderSummary(result)
}
