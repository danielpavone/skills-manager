package config

import (
	"os"
	"strings"
	"testing"
)

func TestWriteConfigTemporaryReportsClosedFileWithOffendingPath(t *testing.T) {
	temporary, err := os.CreateTemp(t.TempDir(), "config-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := temporary.Close(); err != nil {
		t.Fatal(err)
	}
	err = writeConfigTemporary(temporary, []byte("{}"))
	if err == nil || !strings.Contains(err.Error(), temporary.Name()) {
		t.Fatalf("error = %v, want offending temporary path", err)
	}
}
