package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielpavone/skills-manager/internal/app"
	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/config"
	"github.com/danielpavone/skills-manager/internal/project"
)

type FakeProjectUseCase struct {
	result      project.BatchResult
	err         error
	projectPath string
	callCount   int
}

func (f *FakeProjectUseCase) Manage(_ context.Context, projectPath string) (project.BatchResult, error) {
	f.callCount++
	f.projectPath = projectPath
	return f.result, f.err
}

func TestRunnerConfigSetAndShow(t *testing.T) {
	catalogPath := t.TempDir()
	createCLISkill(t, catalogPath)
	store := config.NewJSONStore(filepath.Join(t.TempDir(), "settings"))
	configurator := app.NewCatalogConfigurator(store, catalog.NewFilesystemReader())
	var output, errorOutput bytes.Buffer
	runner := NewRunner(configurator, "dev", &output, &errorOutput)
	if exitCode := runner.Run(context.Background(), []string{"config", "set", catalogPath}); exitCode != SuccessExitCode {
		t.Fatalf("set exit code = %d, want 0; error = %s", exitCode, errorOutput.String())
	}
	if !strings.Contains(output.String(), catalogPath) {
		t.Fatalf("set output = %q, want catalog path", output.String())
	}
	output.Reset()
	if exitCode := runner.Run(context.Background(), []string{"config", "show"}); exitCode != SuccessExitCode {
		t.Fatalf("show exit code = %d, want 0; error = %s", exitCode, errorOutput.String())
	}
	if strings.TrimSpace(output.String()) != catalogPath {
		t.Fatalf("show output = %q, want %q", output.String(), catalogPath)
	}
}

func TestRunnerReturnsDistinctUsageAndFailureCodes(t *testing.T) {
	store := config.NewJSONStore(t.TempDir())
	configurator := app.NewCatalogConfigurator(store, catalog.NewFilesystemReader())
	var output, errorOutput bytes.Buffer
	runner := NewRunner(configurator, "dev", &output, &errorOutput)
	if exitCode := runner.Run(context.Background(), []string{"unknown"}); exitCode != UsageExitCode {
		t.Fatalf("unknown command exit code = %d, want 2", exitCode)
	}
	errorOutput.Reset()
	if exitCode := runner.Run(context.Background(), []string{"config", "show"}); exitCode != FailureExitCode {
		t.Fatalf("missing config exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(errorOutput.String(), "config_missing") {
		t.Fatalf("error output = %q, want config_missing", errorOutput.String())
	}
}

func TestRunnerWritesVersionAndRejectsInteractivePlaceholder(t *testing.T) {
	configurator := app.NewCatalogConfigurator(config.NewJSONStore(t.TempDir()), catalog.NewFilesystemReader())
	var output, errorOutput bytes.Buffer
	runner := NewRunner(configurator, "1.2.3", &output, &errorOutput)
	if exitCode := runner.Run(context.Background(), []string{"--version"}); exitCode != SuccessExitCode {
		t.Fatalf("version exit code = %d, want 0", exitCode)
	}
	if strings.TrimSpace(output.String()) != "1.2.3" {
		t.Fatalf("version output = %q", output.String())
	}
	if exitCode := runner.Run(context.Background(), nil); exitCode != FailureExitCode {
		t.Fatalf("empty args exit code = %d, want 1", exitCode)
	}
}

func TestRunnerDispatchesNoArgumentsToProjectUseCase(t *testing.T) {
	manager := &FakeProjectUseCase{result: project.BatchResult{Results: []project.OperationResult{{
		SkillName: "tdd", Action: project.ActionInstall, Outcome: project.OutcomeInstalled,
	}}}}
	var output, errorOutput bytes.Buffer
	runner := NewRunnerWithProject(
		app.NewCatalogConfigurator(config.NewJSONStore(t.TempDir()), catalog.NewFilesystemReader()),
		manager, "/project", "dev", &output, &errorOutput,
	)

	if exitCode := runner.Run(context.Background(), nil); exitCode != SuccessExitCode {
		t.Fatalf("Run() exit code = %d, want 0; error = %s", exitCode, errorOutput.String())
	}
	if manager.callCount != 1 || manager.projectPath != "/project" {
		t.Fatalf("manager calls = %d, path = %q", manager.callCount, manager.projectPath)
	}
	if !strings.Contains(output.String(), "instaladas:\n- tdd") {
		t.Fatalf("summary = %q, want installed group", output.String())
	}
}

func TestRunnerReturnsSuccessForExplicitCancellation(t *testing.T) {
	manager := &FakeProjectUseCase{err: app.ErrSelectionCanceled}
	var output, errorOutput bytes.Buffer
	runner := NewRunnerWithProject(
		app.NewCatalogConfigurator(config.NewJSONStore(t.TempDir()), catalog.NewFilesystemReader()),
		manager, "/project", "dev", &output, &errorOutput,
	)

	if exitCode := runner.Run(context.Background(), nil); exitCode != SuccessExitCode {
		t.Fatalf("Run() exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(output.String(), "operação cancelada") {
		t.Fatalf("output = %q, want cancellation", output.String())
	}
}

func TestRunnerReturnsFailureAndPrintsPartialBatch(t *testing.T) {
	manager := &FakeProjectUseCase{result: project.BatchResult{
		HasFailures: true,
		Results: []project.OperationResult{
			{SkillName: "tdd", Action: project.ActionInstall, Outcome: project.OutcomeInstalled},
			{SkillName: "broken", Action: project.ActionInstall, Outcome: project.OutcomeFailed, Message: "symlink_permission_denied"},
		},
	}}
	var output, errorOutput bytes.Buffer
	runner := NewRunnerWithProject(
		app.NewCatalogConfigurator(config.NewJSONStore(t.TempDir()), catalog.NewFilesystemReader()),
		manager, "/project", "dev", &output, &errorOutput,
	)

	if exitCode := runner.Run(context.Background(), nil); exitCode != FailureExitCode {
		t.Fatalf("Run() exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(output.String(), "falhas:\n- broken: symlink_permission_denied") {
		t.Fatalf("summary = %q, want failure group", output.String())
	}
}

func createCLISkill(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "tdd")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("# tdd"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
