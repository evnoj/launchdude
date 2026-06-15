package launchctl

import (
	"path/filepath"
	"strings"
)

// Fake is an in-memory Launchctl for tests. Keyed by service target
// (e.g., "gui/501/launchdude.foo"). NOT goroutine-safe.
type Fake struct {
	services map[string]*ServiceState
	// PIDCounter is incremented on each Kickstart to give services unique-ish PIDs.
	PIDCounter int
}

func NewFake() *Fake {
	return &Fake{
		services: map[string]*ServiceState{},
		PIDCounter: 1000,
	}
}

// targetFromPlist derives the gui/<uid>/<label> from a plist path. For the
// fake we don't actually look inside the plist; the test caller arranges
// for Bootstrap to be called with the right domain.
func (f *Fake) targetFromPlist(domain, plistPath string) string {
	base := filepath.Base(plistPath)
	label := strings.TrimSuffix(base, ".plist")
	return domain + "/" + label
}

func (f *Fake) Bootstrap(domain, plistPath string) error {
	target := f.targetFromPlist(domain, plistPath)
	if _, ok := f.services[target]; ok {
		return ErrAlreadyLoaded
	}
	f.services[target] = &ServiceState{
		Loaded:    true,
		PlistPath: plistPath,
		State:     "waiting",
	}
	return nil
}

func (f *Fake) Bootout(target string) error {
	if _, ok := f.services[target]; !ok {
		return ErrNotLoaded
	}
	delete(f.services, target)
	return nil
}

func (f *Fake) Kickstart(target string, killExisting bool) error {
	st, ok := f.services[target]
	if !ok {
		return ErrNotLoaded
	}
	if st.PID != 0 && !killExisting {
		// Already running and not asked to kill — no-op (launchctl behavior).
		return nil
	}
	f.PIDCounter++
	st.PID = f.PIDCounter
	st.State = "running"
	return nil
}

func (f *Fake) Stop(target string) error {
	st, ok := f.services[target]
	if !ok {
		return ErrNotLoaded
	}
	st.PID = 0
	st.State = "exited"
	return nil
}

func (f *Fake) Print(target string) (*ServiceState, error) {
	st, ok := f.services[target]
	if !ok {
		return nil, ErrNotLoaded
	}
	copy := *st
	return &copy, nil
}

func (f *Fake) List(prefix string) ([]ListEntry, error) {
	var out []ListEntry
	for target, st := range f.services {
		label := target
		if i := strings.LastIndex(target, "/"); i >= 0 {
			label = target[i+1:]
		}
		if prefix != "" && !strings.HasPrefix(label, prefix) {
			continue
		}
		out = append(out, ListEntry{
			Label:  label,
			PID:    st.PID,
			Status: st.LastExitCode,
		})
	}
	return out, nil
}
