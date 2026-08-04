package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/launchctl"
	"github.com/evnoj/launchdude/internal/plist"
	"github.com/evnoj/launchdude/internal/service"
)

// setupShowTest isolates HOME/XDG_CONFIG_HOME, wires the cmd-package `manager`
// to a FakeLaunchctl, and returns a buffer wired to showCmd's stdout so tests
// can assert on what the user would have seen.
func setupShowTest(t *testing.T) *bytes.Buffer {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	manager = service.New(launchctl.NewFake())

	stdout := &bytes.Buffer{}
	showCmd.SetOut(stdout)
	showCmd.SetErr(&bytes.Buffer{})

	t.Cleanup(func() {
		manager = nil
		showCmd.SetOut(nil)
		showCmd.SetErr(nil)
	})
	return stdout
}

// installPlist renders the given service to a plist and writes it to the
// service's LaunchAgents path, so drift/orphan states can be exercised.
func installPlist(t *testing.T, name string, svc *config.Service) {
	t.Helper()
	dir, _ := config.LaunchAgentsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := plist.Render(name, svc)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := config.PlistPath(name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestShow_PrintsStateAndProperties: the happy path shows the service label,
// the config path, and the TOML-defined properties.
func TestShow_PrintsStateAndProperties(t *testing.T) {
	stdout := setupShowTest(t)
	writeServiceConfig(t, "demo")

	if err := showCmd.RunE(showCmd, []string{"demo"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{config.Label("demo"), "Keep alive", "Run at load", "Args", "Config"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output; got:\n%s", want, out)
		}
	}
}

// TestShow_ReportsDrift: when the installed plist's embedded hash differs from
// the current TOML, show surfaces the drift warning.
func TestShow_ReportsDrift(t *testing.T) {
	stdout := setupShowTest(t)
	writeServiceConfig(t, "drifty") // exec_args = ["/bin/true"]

	// Install a plist rendered from a *different* config so the embedded hash
	// no longer matches the current TOML.
	installPlist(t, "drifty", &config.Service{
		Name:     "drifty",
		ExecArgs: []string{"/bin/false"},
	})

	if err := showCmd.RunE(showCmd, []string{"drifty"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "config modified since last apply") {
		t.Errorf("expected drift warning in output; got:\n%s", out)
	}
}

// TestShow_UnknownNameErrors: showing a service with neither config nor plist
// should still succeed (it's a valid, if empty, state) — but an invalid name
// must error before any work.
func TestShow_InvalidNameErrors(t *testing.T) {
	setupShowTest(t)

	err := showCmd.RunE(showCmd, []string{"bad name with spaces"})
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

// TestShow_MissingConfigStillPrints: with no config and no plist, show prints
// the state block without crashing and without property rows.
func TestShow_MissingConfigStillPrints(t *testing.T) {
	stdout := setupShowTest(t)

	if err := showCmd.RunE(showCmd, []string{"ghost"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, config.Label("ghost")) {
		t.Errorf("expected label in output; got:\n%s", out)
	}
	if strings.Contains(out, "Keep alive") {
		t.Errorf("property rows should be omitted when config is missing; got:\n%s", out)
	}
}
