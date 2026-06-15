package service

import (
	"os"
	"sort"
	"strings"

	"github.com/evnoj/launchdude/internal/config"
)

// List returns the merged status of all launchdude services across three sources:
//   - TOML configs in services/
//   - Plists in ~/Library/LaunchAgents matching launchdude.*
//   - Loaded services from `launchctl list` matching the launchdude. prefix
//
// The union of names is computed; each entry's Status reflects whichever layers exist.
func (m *Manager) List() ([]*Status, error) {
	names := map[string]struct{}{}

	tomlNames, err := config.ListServiceNames()
	if err != nil {
		return nil, err
	}
	for _, n := range tomlNames {
		names[n] = struct{}{}
	}

	laDir, err := config.LaunchAgentsDir()
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(laDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fn := e.Name()
			if !strings.HasPrefix(fn, config.LabelPrefix) || !strings.HasSuffix(fn, ".plist") {
				continue
			}
			label := strings.TrimSuffix(fn, ".plist")
			if n, ok := config.NameFromLabel(label); ok {
				names[n] = struct{}{}
			}
		}
	}

	// launchctl list is best-effort — if it fails (e.g., launchctl unavailable
	// in tests), we still return what we have from disk.
	if loaded, err := m.lctl.List(config.LabelPrefix); err == nil {
		for _, e := range loaded {
			if n, ok := config.NameFromLabel(e.Label); ok {
				names[n] = struct{}{}
			}
		}
	}

	out := make([]*Status, 0, len(names))
	for n := range names {
		st, err := m.Status(n)
		if err != nil {
			continue
		}
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// AllUserAgents returns labels of every user-domain launchagent visible to
// launchctl (no prefix filter). Used by `list --all`.
func (m *Manager) AllUserAgents() ([]string, error) {
	entries, err := m.lctl.List("")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Label)
	}
	sort.Strings(out)
	return out, nil
}

// pathlessTOMLs and orphanedPlists are convenience helpers used by doctor.
// Kept here next to List so the scanning logic is consistent.

func PathlessTOMLs() ([]string, error) {
	tomlNames, err := config.ListServiceNames()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range tomlNames {
		pp, err := config.PlistPath(n)
		if err != nil {
			continue
		}
		if _, err := os.Stat(pp); os.IsNotExist(err) {
			out = append(out, n)
		}
	}
	return out, nil
}

func OrphanedPlists() ([]string, error) {
	laDir, err := config.LaunchAgentsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(laDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	tomlNames, err := config.ListServiceNames()
	if err != nil {
		return nil, err
	}
	have := map[string]struct{}{}
	for _, n := range tomlNames {
		have[n] = struct{}{}
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fn := e.Name()
		if !strings.HasPrefix(fn, config.LabelPrefix) || !strings.HasSuffix(fn, ".plist") {
			continue
		}
		label := strings.TrimSuffix(fn, ".plist")
		n, ok := config.NameFromLabel(label)
		if !ok {
			continue
		}
		if _, exists := have[n]; !exists {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out, nil
}

