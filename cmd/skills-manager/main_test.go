package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompiledCLIConfiguresAndShowsCatalog(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "skills-manager")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build error = %v\n%s", err, output)
	}
	catalogPath := filepath.Join(t.TempDir(), "catalog")
	if err := os.MkdirAll(filepath.Join(catalogPath, "tdd"), 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(catalogPath, "tdd", "SKILL.md"), []byte("# tdd"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	home := t.TempDir()
	set := exec.Command(binaryPath, "config", "set", catalogPath)
	set.Env = isolatedEnvironment(home)
	if output, err := set.CombinedOutput(); err != nil {
		t.Fatalf("config set error = %v\n%s", err, output)
	}
	show := exec.Command(binaryPath, "config", "show")
	show.Env = isolatedEnvironment(home)
	output, err := show.CombinedOutput()
	if err != nil {
		t.Fatalf("config show error = %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != catalogPath {
		t.Fatalf("config show output = %q, want %q", output, catalogPath)
	}
}

func isolatedEnvironment(home string) []string {
	environment := os.Environ()
	return append(environment, "HOME="+home, "XDG_CONFIG_HOME="+home)
}
