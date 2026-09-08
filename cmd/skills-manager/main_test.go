package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompiledCLIConfiguresAndShowsCatalog(t *testing.T) {
	binaryPath := buildTestCLI(t)
	rootPath, catalogPath := createCommandCatalog(t)
	configRoot := t.TempDir()
	runTestCLI(t, binaryPath, configRoot, "config", "set", rootPath)
	output := runTestCLI(t, binaryPath, configRoot, "config", "show")
	if !strings.Contains(output, "catálogo: "+catalogPath) || !strings.Contains(output, "destino: .agents/skills") {
		t.Fatalf("config show output = %q, want catalog and default target", output)
	}
	settings := filepath.Join(configRoot, "skills-manager", "config.json")
	if runtime.GOOS == "darwin" {
		settings = filepath.Join(configRoot, "Library", "Application Support", "skills-manager", "config.json")
	}
	if _, err := os.Stat(settings); err != nil {
		t.Fatalf("isolated config %q: %v, want temporary file", settings, err)
	}
}

func buildTestCLI(t *testing.T) string {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "skills-manager")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build error = %v\n%s", err, output)
	}
	return binaryPath
}

func createCommandCatalog(t *testing.T) (string, string) {
	t.Helper()
	rootPath := t.TempDir()
	catalogPath := filepath.Join(rootPath, ".agents", "skills")
	if err := os.MkdirAll(filepath.Join(catalogPath, "tdd"), 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(catalogPath, "tdd", "SKILL.md"), []byte("# tdd"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return rootPath, catalogPath
}

func runTestCLI(t *testing.T, binaryPath, configRoot string, args ...string) string {
	t.Helper()
	command := exec.Command(binaryPath, args...)
	command.Env = isolatedEnvironment(configRoot)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI %v error = %v\n%s", args, err, output)
	}
	return string(output)
}

func isolatedEnvironment(configRoot string) []string {
	environment := os.Environ()
	return append(environment, "HOME="+configRoot, "XDG_CONFIG_HOME="+configRoot,
		"APPDATA="+configRoot, "USERPROFILE="+configRoot)
}
