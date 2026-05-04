package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommandSilencesCobraRuntimeErrors(t *testing.T) {
	t.Setenv("PENTA_STORAGE_DB_PATH", filepath.Join(t.TempDir(), "penta.db"))
	cmd := NewPentaCommand()
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"recon", "--session", "missing_session", "1.2.3.4"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected runtime error")
	}
	if out.String() != "" {
		t.Fatalf("runtime error wrote to stdout: %q", out.String())
	}
	if errOut.String() != "" {
		t.Fatalf("cobra printed runtime error despite SilenceErrors: %q", errOut.String())
	}
	if strings.Contains(err.Error(), "Usage:") {
		t.Fatalf("runtime error included usage: %v", err)
	}
}

func TestRootCommandHelpStillShowsUsageAfterSilencingErrors(t *testing.T) {
	t.Setenv("PENTA_STORAGE_DB_PATH", filepath.Join(t.TempDir(), "penta.db"))
	cmd := NewPentaCommand()
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"recon", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("help output missing usage: %q", out.String())
	}
	if errOut.String() != "" {
		t.Fatalf("help wrote to stderr: %q", errOut.String())
	}
}
