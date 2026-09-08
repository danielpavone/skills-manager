package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/danielpavone/skills-manager/internal/agentdir"
	"github.com/danielpavone/skills-manager/internal/app"
	"github.com/danielpavone/skills-manager/internal/config"
	"github.com/danielpavone/skills-manager/internal/project"
	"github.com/danielpavone/skills-manager/internal/summary"
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
	fmt.Fprintln(r.output, summary.RenderSummary(result))
}

func (r Runner) runConfig(ctx context.Context, args []string) int {
	if len(args) > 0 && args[0] == "set" {
		return r.runConfigSet(ctx, args[1:])
	}
	if len(args) == 1 && args[0] == "show" {
		return r.runConfigShow(ctx)
	}
	return r.writeUsage(configUsage())
}

func (r Runner) runConfigSet(ctx context.Context, args []string) int {
	catalogPath, target, err := parseConfigSetArgs(args)
	if err != nil {
		return r.writeUsage(err.Error())
	}
	configured, err := r.catalog.SetForTarget(ctx, catalogPath, target)
	if err != nil {
		return r.writeError(err.Error())
	}
	r.writeConfiguration(configured)
	return SuccessExitCode
}

func (r Runner) runConfigShow(ctx context.Context) int {
	configured, err := r.catalog.Show(ctx)
	if err != nil {
		return r.writeError(err.Error())
	}
	r.writeConfiguration(configured)
	return SuccessExitCode
}

func parseConfigSetArgs(args []string) (string, agentdir.Directory, error) {
	if len(args) == 1 {
		return args[0], agentdir.Agents, nil
	}
	if len(args) != 3 || args[1] != "--target" {
		return "", "", errors.New(configUsage())
	}
	target, err := agentdir.Parse(args[2])
	if err != nil {
		return "", "", err
	}
	return args[0], target, nil
}

func (r Runner) writeConfiguration(configured config.CatalogConfig) {
	fmt.Fprintf(r.output, "catálogo: %s\n", configured.CatalogPath)
	fmt.Fprintf(r.output, "destino: %s/skills\n", configured.EffectiveTargetDirectory())
}

func configUsage() string {
	return "uso: skills-manager config set <caminho> [--target .agents|.claude|.devin] | skills-manager config show"
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
