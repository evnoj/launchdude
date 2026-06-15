package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/launchctl"
	"github.com/evnoj/launchdude/internal/plist"
)

type Manager struct {
	lctl launchctl.Launchctl
}

func New(lctl launchctl.Launchctl) *Manager {
	return &Manager{lctl: lctl}
}

// Apply renders the plist from the TOML config and reconciles it with launchd.
// If the service is currently loaded, it boots it out and back in so the new
// plist takes effect. If not loaded, the plist is just written to disk.
func (m *Manager) Apply(name string) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	svc, err := config.LoadService(name)
	if err != nil {
		return err
	}

	rendered, err := plist.Render(name, svc)
	if err != nil {
		return err
	}

	plistPath, err := config.PlistPath(name)
	if err != nil {
		return err
	}
	if err := config.EnsureDir(filepath.Dir(plistPath)); err != nil {
		return err
	}
	logsDir, err := config.LogsDir()
	if err != nil {
		return err
	}
	if err := config.EnsureDir(logsDir); err != nil {
		return err
	}

	// Short-circuit if the on-disk plist already matches what we'd render.
	// Without this, every `enable` call would churn the service (bootout +
	// bootstrap restarts it even if nothing changed). Hash-based drift
	// detection sees bytes.Equal as "in sync" — no mtime touch needed.
	if existing, err := os.ReadFile(plistPath); err == nil && bytes.Equal(existing, rendered) {
		return nil
	}

	target, err := config.ServiceTarget(name)
	if err != nil {
		return err
	}
	_, printErr := m.lctl.Print(target)
	wasLoaded := printErr == nil
	if printErr != nil && !errors.Is(printErr, launchctl.ErrNotLoaded) {
		return printErr
	}

	if wasLoaded {
		if err := m.lctl.Bootout(target); err != nil && !errors.Is(err, launchctl.ErrNotLoaded) {
			return err
		}
	}

	if err := os.WriteFile(plistPath, rendered, 0o644); err != nil {
		return err
	}

	if wasLoaded {
		domain, err := config.GUIDomain()
		if err != nil {
			return err
		}
		if err := m.lctl.Bootstrap(domain, plistPath); err != nil && !errors.Is(err, launchctl.ErrAlreadyLoaded) {
			return err
		}
	}
	return nil
}

type EnableOpts struct {
	NoStart bool
}

// Enable makes the service "on": applies pending TOML changes, ensures it's
// registered with launchd, and starts it (unless opts.NoStart). Idempotent —
// running twice does nothing the second time. This is the bug serviceman had.
func (m *Manager) Enable(name string, opts EnableOpts) error {
	if err := m.Apply(name); err != nil {
		return err
	}
	target, err := config.ServiceTarget(name)
	if err != nil {
		return err
	}
	plistPath, err := config.PlistPath(name)
	if err != nil {
		return err
	}
	domain, err := config.GUIDomain()
	if err != nil {
		return err
	}

	state, err := m.lctl.Print(target)
	switch {
	case errors.Is(err, launchctl.ErrNotLoaded):
		if berr := m.lctl.Bootstrap(domain, plistPath); berr != nil && !errors.Is(berr, launchctl.ErrAlreadyLoaded) {
			return fmt.Errorf("bootstrap: %w", berr)
		}
		state, err = m.lctl.Print(target)
		if err != nil {
			return fmt.Errorf("re-query after bootstrap: %w", err)
		}
	case err != nil:
		return err
	}

	if opts.NoStart {
		return nil
	}
	if state.PID == 0 {
		if err := m.lctl.Kickstart(target, false); err != nil {
			return fmt.Errorf("kickstart: %w", err)
		}
	}
	return nil
}

// Start kickstarts an installed-and-loaded service. Errors if the service isn't
// loaded — by design, start is a dumb toggle on the current installation and
// won't auto-install. Use `enable` to bootstrap from scratch.
func (m *Manager) Start(name string) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	target, err := config.ServiceTarget(name)
	if err != nil {
		return err
	}
	state, err := m.lctl.Print(target)
	if errors.Is(err, launchctl.ErrNotLoaded) {
		return fmt.Errorf("service %q is not loaded; run `launchdude enable %s` first", name, name)
	}
	if err != nil {
		return err
	}
	if state.PID > 0 {
		return nil // already running
	}
	return m.lctl.Kickstart(target, false)
}

// Stop kills a running service. No-op if already stopped, errors if not loaded.
func (m *Manager) Stop(name string) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	target, err := config.ServiceTarget(name)
	if err != nil {
		return err
	}
	state, err := m.lctl.Print(target)
	if errors.Is(err, launchctl.ErrNotLoaded) {
		return fmt.Errorf("service %q is not loaded", name)
	}
	if err != nil {
		return err
	}
	if state.PID == 0 {
		return nil // already stopped
	}
	return m.lctl.Stop(target)
}

// Restart reconciles pending TOML changes and ensures the service is running
// with a fresh process. Equivalent to: apply, ensure loaded, kickstart-with-kill.
func (m *Manager) Restart(name string) error {
	if err := m.Apply(name); err != nil {
		return err
	}
	target, err := config.ServiceTarget(name)
	if err != nil {
		return err
	}
	plistPath, err := config.PlistPath(name)
	if err != nil {
		return err
	}
	domain, err := config.GUIDomain()
	if err != nil {
		return err
	}

	_, err = m.lctl.Print(target)
	if errors.Is(err, launchctl.ErrNotLoaded) {
		if berr := m.lctl.Bootstrap(domain, plistPath); berr != nil && !errors.Is(berr, launchctl.ErrAlreadyLoaded) {
			return fmt.Errorf("bootstrap: %w", berr)
		}
	} else if err != nil {
		return err
	}
	return m.lctl.Kickstart(target, true) // killExisting=true
}

type DisableOpts struct {
	NoStop bool
}

// Disable boots the service out of launchd. By default stops it first if running.
// Idempotent — running on an already-disabled service is a no-op.
func (m *Manager) Disable(name string, opts DisableOpts) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	target, err := config.ServiceTarget(name)
	if err != nil {
		return err
	}
	state, err := m.lctl.Print(target)
	if errors.Is(err, launchctl.ErrNotLoaded) {
		return nil // already disabled
	}
	if err != nil {
		return err
	}
	if !opts.NoStop && state.PID > 0 {
		if err := m.lctl.Stop(target); err != nil && !errors.Is(err, launchctl.ErrNotLoaded) {
			return err
		}
	}
	if err := m.lctl.Bootout(target); err != nil && !errors.Is(err, launchctl.ErrNotLoaded) {
		return err
	}
	return nil
}

type ImportOpts struct {
	Force bool // overwrite existing TOML if present
}

// Import reads the launchdude.<name>.plist file and writes an equivalent TOML
// service config. Used to recover from a deleted TOML or adopt an existing plist.
// If a TOML already exists, requires opts.Force.
func (m *Manager) Import(name string, opts ImportOpts) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	plistPath, err := config.PlistPath(name)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(plistPath)
	if err != nil {
		return fmt.Errorf("read plist %s: %w", plistPath, err)
	}
	svc, _, err := plist.Parse(data)
	if err != nil {
		return err
	}

	// Strip auto-routed log paths so the TOML stays clean and benefits from
	// future default changes. Only clear if they match our default exactly.
	defStdout, _ := config.DefaultStdoutPath(name)
	if svc.StdoutPath == defStdout {
		svc.StdoutPath = ""
	}
	defStderr, _ := config.DefaultStderrPath(name)
	if svc.StderrPath == defStderr {
		svc.StderrPath = ""
	}

	tomlPath, err := config.ServiceConfigPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(tomlPath); err == nil && !opts.Force {
		return fmt.Errorf("config already exists at %s; use --force to overwrite", tomlPath)
	}
	return config.SaveService(name, svc)
}

// Delete fully removes a known service: disable, hard-remove the generated
// plist, and send the TOML config to the macOS Trash. Errors if no TOML
// exists for `name` — strict by design so typos don't silently no-op against
// the wrong service. For orphan cleanup (plist exists, no TOML), use
// DeleteOrphan from the doctor flow.
func (m *Manager) Delete(name string) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	cfgPath, err := config.ServiceConfigPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("no service named %q (no config at %s)", name, cfgPath)
	} else if err != nil {
		return err
	}
	if err := m.removeLaunchAgent(name); err != nil {
		return err
	}
	return config.TrashServiceConfig(name)
}

// DeleteOrphan removes a launchagent whose TOML config is already gone — the
// doctor flow's "delete the orphan" action. Disables and removes the plist;
// does not touch the (non-existent) TOML.
func (m *Manager) DeleteOrphan(name string) error {
	if err := config.ValidateName(name); err != nil {
		return err
	}
	return m.removeLaunchAgent(name)
}

// removeLaunchAgent is the shared subset of both delete paths: stop the
// service if running, bootout, and hard-remove the plist from
// ~/Library/LaunchAgents. Idempotent.
func (m *Manager) removeLaunchAgent(name string) error {
	if err := m.Disable(name, DisableOpts{}); err != nil {
		return err
	}
	plistPath, err := config.PlistPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

