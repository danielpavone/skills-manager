package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/danielpavone/skills-manager/internal/app"
	"github.com/danielpavone/skills-manager/internal/project"
)

const (
	SuccessExitCode = 0
	FailureExitCode = 1
	UsageExitCode   = 2
)

type Runner struct {
	catalog     app.CatalogConfigurator
	project     ProjectUseCase
	projectPath string
	version     string
	output      io.Writer
	errorOutput io.Writer
}

type ProjectUseCase interface {
	Manage(ctx context.Context, projectPath string) (project.BatchResult, error)
}

func NewRunner(catalog app.CatalogConfigurator, version string, output, errorOutput io.Writer, manager ...ProjectUseCase) Runner {
	runner := Runner{catalog: catalog, version: version, output: output, errorOutput: errorOutput}
	if len(manager) > 0 {
		runner.project = manager[0]
	}
	return runner
}

func NewRunnerWithProject(catalog app.CatalogConfigurator, manager ProjectUseCase, projectPath, version string, output, errorOutput io.Writer) Runner {
	return Runner{
		catalog: catalog, project: manager, projectPath: projectPath,
		version: version, output: output, errorOutput: errorOutput,
	}
}

func (r Runner) Run(ctx context.Context, args []string) int {
	if len(args) == 1 && args[0] == "--version" {
		return r.writeVersion()
	}
	if len(args) == 0 {
		return r.runProject(ctx)
	}
	if args[0] != "config" {
		return r.writeUsage(fmt.Sprintf("comando desconhecido %q", args[0]))
	}
	return r.runConfig(ctx, args[1:])
}

func (r Runner) runProject(ctx context.Context) int {
	if r.project == nil {
		return r.writeError("gerenciamento de projeto indisponível: SelectionUI configurada no composition root")
	}
	result, err := r.project.Manage(ctx, r.projectPath)
	if err != nil {
		return r.writeProjectError(err)
	}
	return r.writeProjectResult(result)
}

func (r Runner) writeProjectError(err error) int {
	if errors.Is(err, app.ErrSelectionCanceled) {
		fmt.Fprintln(r.output, "operação cancelada")
		return SuccessExitCode
	}
	return r.writeError(err.Error())
}

func (r Runner) writeProjectResult(result project.BatchResult) int {
	r.writeBatchSummary(result)
	if result.HasFailures {
		return FailureExitCode
	}
	return SuccessExitCode
}

func (r Runner) writeBatchSummary(result project.BatchResult) {
	r.writeOutcomeGroup(result.Results, project.OutcomeInstalled, "instaladas")
	r.writeOutcomeGroup(result.Results, project.OutcomeRemoved, "removidas")
	r.writeOutcomeGroup(result.Results, project.OutcomeUnchanged, "inalteradas")
	r.writeOutcomeGroup(result.Results, project.OutcomeFailed, "falhas")
}

func (r Runner) writeOutcomeGroup(results []project.OperationResult, outcome project.OperationOutcome, label string) {
	items := matchingOutcomes(results, outcome)
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(r.output, "%s:\n", label)
	r.writeOutcomeItems(items)
}

func matchingOutcomes(results []project.OperationResult, outcome project.OperationOutcome) []project.OperationResult {
	items := make([]project.OperationResult, 0, len(results))
	for _, result := range results {
		if result.Outcome == outcome {
			items = append(items, result)
		}
	}
	return items
}

func (r Runner) writeOutcomeItems(items []project.OperationResult) {
	for _, item := range items {
		if item.Message == "" {
			fmt.Fprintf(r.output, "- %s\n", item.SkillName)
			continue
		}
		fmt.Fprintf(r.output, "- %s: %s\n", item.SkillName, item.Message)
	}
}

func (r Runner) runConfig(ctx context.Context, args []string) int {
	if len(args) == 2 && args[0] == "set" {
		configured, err := r.catalog.Set(ctx, args[1])
		if err != nil {
			return r.writeError(err.Error())
		}
		fmt.Fprintf(r.output, "catálogo configurado: %s\n", configured.CatalogPath)
		return SuccessExitCode
	}
	if len(args) == 1 && args[0] == "show" {
		configured, err := r.catalog.Show(ctx)
		if err != nil {
			return r.writeError(err.Error())
		}
		fmt.Fprintf(r.output, "%s\n", configured.CatalogPath)
		return SuccessExitCode
	}
	return r.writeUsage("uso: skills-manager config set <caminho> | skills-manager config show")
}

func (r Runner) writeVersion() int {
	fmt.Fprintln(r.output, r.version)
	return SuccessExitCode
}

func (r Runner) writeError(message string) int {
	fmt.Fprintf(r.errorOutput, "skills-manager: %s\n", message)
	return FailureExitCode
}

func (r Runner) writeUsage(message string) int {
	fmt.Fprintf(r.errorOutput, "skills-manager: %s\n", message)
	return UsageExitCode
}
