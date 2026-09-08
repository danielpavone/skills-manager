package main

import (
	"context"
	"fmt"
	"os"

	"github.com/danielpavone/skills-manager/internal/app"
	"github.com/danielpavone/skills-manager/internal/catalog"
	"github.com/danielpavone/skills-manager/internal/cli"
	"github.com/danielpavone/skills-manager/internal/config"
	"github.com/danielpavone/skills-manager/internal/project"
	"github.com/danielpavone/skills-manager/internal/tui"
)

var version = "dev"

func main() {
	store, err := config.NewUserJSONStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "skills-manager: %s\n", err)
		os.Exit(cli.FailureExitCode)
	}
	reader := catalog.NewFilesystemReader()
	configurator := app.NewCatalogConfigurator(store, reader)
	links := project.NewFilesystemLinks()
	selection := tui.NewSelectionUI(os.Stdin, os.Stdout)
	manager := app.NewManageProject(store, reader, links, selection)
	projectPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "skills-manager: não foi possível localizar o projeto atual: %s\n", err)
		os.Exit(cli.FailureExitCode)
	}
	runner := cli.NewRunnerWithProject(configurator, manager, projectPath, version, os.Stdout, os.Stderr)
	os.Exit(runner.Run(context.Background(), os.Args[1:]))
}
