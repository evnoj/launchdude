package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/launchctl"
	"github.com/evnoj/launchdude/internal/service"
)

// setupDeleteTest isolates HOME/XDG_CONFIG_HOME, wires the cmd-package
// `manager` to a FakeLaunchctl, and resets/restores command flag state.
// Returns buffers wired to deleteCmd's output streams so tests can assert
// on what the user would have seen.
func setupDeleteTest(t *testing.T) (stdout, stderr *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	manager = service.New(launchctl.NewFake())
	deleteForce = false

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	deleteCmd.SetOut(stdout)
	deleteCmd.SetErr(stderr)

	t.Cleanup(func() {
		manager = nil
		deleteForce = false
		deleteCmd.SetOut(nil)
		deleteCmd.SetErr(nil)
	})
	return stdout, stderr
}

// writeServiceConfig writes a minimal valid TOML for `name` under the isolated
// services dir.
func writeServiceConfig(t *testing.T, name string) {
	t.Helper()
	dir, _ := config.ServicesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, _ := config.ServiceConfigPath(name)
	body := `name = "` + name + `"
exec_args = ["/bin/true"]
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDelete_UnknownNameErrorsBeforePrompt is the regression test for the
// user-reported issue: `delete` must check file existence before showing
// the confirmation prompt, so the user doesn't get prompted to confirm
// deletion of something that doesn't exist.
func TestDelete_UnknownNameErrorsBeforePrompt(t *testing.T) {
	_, stderr := setupDeleteTest(t)

	err := deleteCmd.RunE(deleteCmd, []string{"ghost"})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	if !strings.Contains(err.Error(), "no service") {
		t.Errorf("error message: %v", err)
	}
	// The "delete X? [y/N]" prompt must not appear when the service didn't exist.
	if strings.Contains(stderr.String(), "delete ") {
		t.Errorf("prompt should not have been shown; stderr: %q", stderr.String())
	}
}

// TestDelete_UnknownNameErrorsEvenWithForce: --force changes the prompt
// behavior, not the existence check. Trying to delete a nonexistent service
// still fails regardless.
func TestDelete_UnknownNameErrorsEvenWithForce(t *testing.T) {
	setupDeleteTest(t)
	deleteForce = true

	err := deleteCmd.RunE(deleteCmd, []string{"ghost"})
	if err == nil {
		t.Fatal("expected error even with --force")
	}
	if !strings.Contains(err.Error(), "no service") {
		t.Errorf("error message: %v", err)
	}
}

// TestDelete_ExistingNameWithForce confirms the happy path still works:
// when the TOML exists, --force proceeds without prompting.
func TestDelete_ExistingNameWithForce(t *testing.T) {
	stdout, stderr := setupDeleteTest(t)
	writeServiceConfig(t, "real")
	deleteForce = true

	if err := deleteCmd.RunE(deleteCmd, []string{"real"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "deleted") {
		t.Errorf("expected 'deleted' in stdout; got %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "delete ") {
		t.Errorf("--force should suppress the prompt; stderr: %q", stderr.String())
	}

	// And the config has been moved out of services/.
	cfgPath, _ := config.ServiceConfigPath("real")
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config should be moved out of services/; stat err: %v", err)
	}
}

// TestDelete_InvalidNameRejected covers a sibling early-return: an obviously
// invalid name should fail validation before any filesystem work or prompts.
func TestDelete_InvalidNameRejected(t *testing.T) {
	_, stderr := setupDeleteTest(t)

	err := deleteCmd.RunE(deleteCmd, []string{"bad name with spaces"})
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if strings.Contains(stderr.String(), "delete ") {
		t.Errorf("prompt should not appear for invalid name; stderr: %q", stderr.String())
	}
}
