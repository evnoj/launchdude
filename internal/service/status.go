package service

import (
	"errors"
	"os"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/launchctl"
	"github.com/evnoj/launchdude/internal/plist"
)

type Status struct {
	Name string

	ConfigPath   string
	ConfigExists bool

	PlistPath    string
	PlistExists  bool
	Drifted      bool // TOML newer than plist

	Loaded       bool
	PID          int
	State        string // "running", "exited", "waiting", etc.
	LastExitCode int

	StdoutPath string
	StderrPath string
	WorkingDir string

	Program string // executable from the loaded plist (if loaded)
}

func (s *Status) Running() bool { return s.PID > 0 }

// Status reports the full state of a service across all 4 layers
// (config / installed plist / loaded / running).
func (m *Manager) Status(name string) (*Status, error) {
	if err := config.ValidateName(name); err != nil {
		return nil, err
	}
	st := &Status{Name: name}

	cfgPath, err := config.ServiceConfigPath(name)
	if err != nil {
		return nil, err
	}
	st.ConfigPath = cfgPath
	var svc *config.Service
	if _, statErr := os.Stat(cfgPath); statErr == nil {
		st.ConfigExists = true
		svc, _ = config.LoadService(name) // tolerate unparseable; drift just won't be claimed
	}

	plistPath, err := config.PlistPath(name)
	if err != nil {
		return nil, err
	}
	st.PlistPath = plistPath
	if plistBytes, err := os.ReadFile(plistPath); err == nil {
		st.PlistExists = true
		// Drift = the installed plist's embedded LaunchdudeHash differs from
		// the canonical hash of the parsed TOML. If the plist has no
		// LaunchdudeHash (e.g., adopted from outside launchdude), or the TOML
		// fails to parse, we don't claim drift.
		if svc != nil {
			if installedHash, err := plist.ParseHash(plistBytes); err == nil && installedHash != "" {
				if plist.HashConfig(svc) != installedHash {
					st.Drifted = true
				}
			}
		}
	}

	target, err := config.ServiceTarget(name)
	if err != nil {
		return nil, err
	}
	state, err := m.lctl.Print(target)
	switch {
	case err == nil:
		st.Loaded = true
		st.PID = state.PID
		st.State = state.State
		st.LastExitCode = state.LastExitCode
		st.Program = state.Program
	case errors.Is(err, launchctl.ErrNotLoaded):
		// stays not-loaded; not an error
	default:
		return nil, err
	}

	if svc != nil {
		st.StdoutPath, _ = svc.ResolveStdoutPath(name)
		st.StderrPath, _ = svc.ResolveStderrPath(name)
		st.WorkingDir, _ = svc.ResolveWorkingDir()
	}
	return st, nil
}
