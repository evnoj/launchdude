package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/launchctl"
)

// setupManager isolates HOME/XDG_CONFIG_HOME to a temp dir and wires a Manager
// to an in-memory FakeLaunchctl. Returns the manager and fake for test assertions.
func setupManager(t *testing.T) (*Manager, *launchctl.Fake) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	fake := launchctl.NewFake()
	return New(fake), fake
}

// writeConfig writes a valid TOML config under services/<name>.toml.
func writeConfig(t *testing.T, name string, body string) {
	t.Helper()
	dir, _ := config.ServicesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, _ := config.ServiceConfigPath(name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// validTOML returns a minimal-but-valid TOML body for `name` that the Manager
// can render and the Fake can run.
func validTOML(name string) string {
	return `name = "` + name + `"
exec_args = ["/bin/sh", "-c", "sleep 60"]
keep_alive = true
run_at_load = true
`
}

func TestEnable_FirstCallBootstrapsAndStarts(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))

	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	target, _ := config.ServiceTarget("foo")
	st, err := fake.Print(target)
	if err != nil {
		t.Fatalf("Print: %v", err)
	}
	if st.PID == 0 {
		t.Error("expected service to be running after Enable")
	}
}

// TestEnable_Idempotent is the canonical "serviceman bug fixed" test. Running
// Enable twice on an already-enabled service must succeed (no leaked
// ErrAlreadyLoaded) and must not churn the running process.
func TestEnable_Idempotent(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))

	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatalf("first Enable: %v", err)
	}
	target, _ := config.ServiceTarget("foo")
	st, _ := fake.Print(target)
	firstPID := st.PID
	if firstPID == 0 {
		t.Fatal("first Enable should have started the service")
	}

	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatalf("second Enable returned error (this was the serviceman bug): %v", err)
	}
	st, _ = fake.Print(target)
	if st.PID != firstPID {
		t.Errorf("second Enable should be a no-op; PID changed %d -> %d", firstPID, st.PID)
	}
}

func TestEnable_NoStartRegistersButDoesNotStart(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))

	if err := m.Enable("foo", EnableOpts{NoStart: true}); err != nil {
		t.Fatal(err)
	}
	target, _ := config.ServiceTarget("foo")
	st, err := fake.Print(target)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Loaded {
		t.Error("--no-start should still register the service")
	}
	if st.PID != 0 {
		t.Errorf("--no-start should NOT kickstart; got PID %d", st.PID)
	}
}

func TestApply_NoOpShortCircuitWhenBytesMatch(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))

	// Enable installs the plist and starts the service.
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	target, _ := config.ServiceTarget("foo")
	st, _ := fake.Print(target)
	firstPID := st.PID

	// Apply with no config change: should short-circuit (no churn).
	if err := m.Apply("foo"); err != nil {
		t.Fatal(err)
	}
	st, _ = fake.Print(target)
	if st.PID != firstPID {
		t.Errorf("Apply on unchanged config should not restart; PID %d -> %d", firstPID, st.PID)
	}
}

func TestApply_ReloadsWhenConfigChanges(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	target, _ := config.ServiceTarget("foo")
	st, _ := fake.Print(target)
	firstPID := st.PID

	// Edit the config to a different exec — bytes will differ.
	writeConfig(t, "foo", `name = "foo"
exec_args = ["/bin/echo", "different"]
keep_alive = true
`)
	if err := m.Apply("foo"); err != nil {
		t.Fatal(err)
	}
	st, _ = fake.Print(target)
	if !st.Loaded {
		t.Error("service should still be loaded after Apply with new config")
	}
	// PID may or may not change in the fake (it doesn't auto-kickstart on
	// bootstrap), but at minimum the bootout-and-bootstrap cycle must have
	// reset the running PID to zero.
	if st.PID == firstPID {
		t.Errorf("Apply on changed config should have reloaded; PID unchanged at %d", firstPID)
	}
}

func TestDisable_StopsAndBootsOut(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}

	if err := m.Disable("foo", DisableOpts{}); err != nil {
		t.Fatal(err)
	}
	target, _ := config.ServiceTarget("foo")
	_, err := fake.Print(target)
	if !errors.Is(err, launchctl.ErrNotLoaded) {
		t.Errorf("after Disable, Print should return ErrNotLoaded; got %v", err)
	}
}

func TestDisable_IdempotentOnAlreadyDisabled(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	// Never enabled, so it's already in the "disabled" state.
	if err := m.Disable("foo", DisableOpts{}); err != nil {
		t.Errorf("Disable on already-disabled service should be a no-op, got %v", err)
	}
}

func TestStart_ErrorsWhenNotLoaded(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	err := m.Start("foo")
	if err == nil {
		t.Fatal("Start on unloaded service should error")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("error should mention 'not loaded': %v", err)
	}
}

func TestStop_ErrorsWhenNotLoaded(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	err := m.Stop("foo")
	if err == nil {
		t.Fatal("Stop on unloaded service should error")
	}
}

func TestRestart_GivesFreshProcess(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	target, _ := config.ServiceTarget("foo")
	st, _ := fake.Print(target)
	firstPID := st.PID

	if err := m.Restart("foo"); err != nil {
		t.Fatal(err)
	}
	st, _ = fake.Print(target)
	if st.PID == firstPID {
		t.Errorf("Restart should produce a new PID; still %d", firstPID)
	}
	if st.PID == 0 {
		t.Error("Restart should leave a running process")
	}
}

func TestDelete_StrictRejectsUnknownName(t *testing.T) {
	m, _ := setupManager(t)
	err := m.Delete("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	if !strings.Contains(err.Error(), "no service") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelete_SendsTOMLToTrash(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}

	cfgPath, _ := config.ServiceConfigPath("foo")
	if err := m.Delete("foo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Error("config file should be moved out of services/")
	}
	home, _ := os.UserHomeDir()
	trashed := filepath.Join(home, ".Trash", "foo.toml")
	if _, err := os.Stat(trashed); err != nil {
		t.Errorf("expected config at ~/.Trash/foo.toml: %v", err)
	}
}

func TestDeleteOrphan_NoTOMLRequired(t *testing.T) {
	m, fake := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	// Simulate orphan: remove TOML directly.
	cfgPath, _ := config.ServiceConfigPath("foo")
	os.Remove(cfgPath)

	if err := m.DeleteOrphan("foo"); err != nil {
		t.Fatalf("DeleteOrphan should succeed without TOML, got %v", err)
	}
	target, _ := config.ServiceTarget("foo")
	if _, err := fake.Print(target); !errors.Is(err, launchctl.ErrNotLoaded) {
		t.Errorf("after DeleteOrphan, service should be unloaded; got %v", err)
	}
}

func TestStatus_PendingWhenConfigExistsButNoPlist(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	st, err := m.Status("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !st.ConfigExists {
		t.Error("ConfigExists should be true")
	}
	if st.PlistExists {
		t.Error("PlistExists should be false before Enable/Apply")
	}
	if st.Loaded {
		t.Error("Loaded should be false")
	}
}

func TestStatus_OrphanWhenPlistExistsButNoConfig(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	// Remove TOML to create orphan state.
	cfgPath, _ := config.ServiceConfigPath("foo")
	os.Remove(cfgPath)

	st, err := m.Status("foo")
	if err != nil {
		t.Fatal(err)
	}
	if st.ConfigExists {
		t.Error("ConfigExists should be false (orphan)")
	}
	if !st.PlistExists {
		t.Error("PlistExists should still be true")
	}
}

func TestStatus_DriftWhenConfigChanges(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}

	// Substantive edit: different exec → different hash.
	writeConfig(t, "foo", `name = "foo"
exec_args = ["/bin/different"]
keep_alive = true
`)
	st, err := m.Status("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !st.Drifted {
		t.Error("expected Drifted=true after substantive config edit")
	}
}

func TestStatus_NoDriftFromCosmeticEdit(t *testing.T) {
	// The canonical-hash regression: cosmetic edits (comments, blank lines,
	// env map reorder) must NOT trigger drift.
	m, _ := setupManager(t)
	writeConfig(t, "foo", `name = "foo"
exec_args = ["/bin/sh", "-c", "sleep 60"]
keep_alive = true
[env]
ALPHA = "1"
BRAVO = "2"
`)
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}

	// Rewrite with comment, blank lines, and reordered env keys.
	writeConfig(t, "foo", `# a comment
name = "foo"

exec_args = ["/bin/sh", "-c", "sleep 60"]
keep_alive = true

[env]
BRAVO = "2"
ALPHA = "1"
`)
	st, err := m.Status("foo")
	if err != nil {
		t.Fatal(err)
	}
	if st.Drifted {
		t.Error("cosmetic edits should NOT trigger drift")
	}
}

func TestImport_ReadsPlistAndWritesTOML(t *testing.T) {
	m, _ := setupManager(t)
	// First enable a service so a plist exists on disk.
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	// Remove the TOML to simulate an orphan that we want to recover.
	cfgPath, _ := config.ServiceConfigPath("foo")
	os.Remove(cfgPath)

	if err := m.Import("foo", ImportOpts{}); err != nil {
		t.Fatalf("Import: %v", err)
	}
	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("TOML should be recreated by Import: %v", err)
	}

	// Recovered TOML should be loadable.
	svc, err := config.LoadService("foo")
	if err != nil {
		t.Fatalf("recovered TOML failed to load: %v", err)
	}
	if svc.Name != "foo" {
		t.Errorf("imported Name: %q", svc.Name)
	}
}

func TestImport_RefusesWhenTOMLExists(t *testing.T) {
	m, _ := setupManager(t)
	writeConfig(t, "foo", validTOML("foo"))
	if err := m.Enable("foo", EnableOpts{}); err != nil {
		t.Fatal(err)
	}
	// TOML still on disk.
	err := m.Import("foo", ImportOpts{})
	if err == nil {
		t.Fatal("Import without --force should refuse to clobber existing TOML")
	}
}
