package launchctl

import (
	"testing"
)

// printRunning is a synthetic `launchctl print` fixture mimicking the real
// curly-brace nested key/value format launchctl emits, with realistic field
// names but throwaway values. The parser only cares about a handful of keys;
// the rest of the file is intentionally messy to ensure unparsed sections
// don't trip us up.
const printRunning = `gui/100/launchdude.example = {
	active count = 1
	path = /tmp/fake/LaunchAgents/launchdude.example.plist
	type = LaunchAgent
	state = running

	program = /tmp/fake/bin/example
	arguments = {
		/tmp/fake/bin/example
		--flag
		value
	}

	stdout path = /tmp/fake/Logs/example.out.log
	stderr path = /tmp/fake/Logs/example.err.log

	domain = gui/100 [123456]
	pid = 11111
	last exit code = (never exited)
	runs = 1
}
`

// printExited is a synthetic fixture showing a service that has exited with a
// non-zero code. Verified field name "last exit code" matches launchctl output.
const printExited = `gui/100/launchdude.failer = {
	path = /tmp/fake/LaunchAgents/launchdude.failer.plist
	type = LaunchAgent
	state = exited

	program = /tmp/fake/bin/failer
	pid = 0
	last exit code = 1
}
`

const listOutput = `PID	Status	Label
-	0	com.example.headless
22222	0	com.example.running
33333	0	launchdude.alpha
-	1	launchdude.beta
`

func TestParsePrint_Running(t *testing.T) {
	st := parsePrint(printRunning)
	if !st.Loaded {
		t.Error("Loaded should be true after parsePrint")
	}
	if st.PID != 11111 {
		t.Errorf("PID: want 11111, got %d", st.PID)
	}
	if st.State != "running" {
		t.Errorf("State: want running, got %q", st.State)
	}
	if st.Program != "/tmp/fake/bin/example" {
		t.Errorf("Program: want /tmp/fake/bin/example, got %q", st.Program)
	}
	if st.PlistPath != "/tmp/fake/LaunchAgents/launchdude.example.plist" {
		t.Errorf("PlistPath: got %q", st.PlistPath)
	}
	// "(never exited)" doesn't parse as int — LastExitCode stays at zero value.
	if st.LastExitCode != 0 {
		t.Errorf("LastExitCode: want 0 (never exited), got %d", st.LastExitCode)
	}
}

func TestParsePrint_Exited(t *testing.T) {
	st := parsePrint(printExited)
	if st.State != "exited" {
		t.Errorf("State: want exited, got %q", st.State)
	}
	if st.PID != 0 {
		t.Errorf("PID: want 0, got %d", st.PID)
	}
	if st.LastExitCode != 1 {
		t.Errorf("LastExitCode: want 1, got %d", st.LastExitCode)
	}
}

func TestParseList_NoPrefix(t *testing.T) {
	entries := parseList(listOutput, "")
	// Header row should be skipped; expect 4 entries.
	if len(entries) != 4 {
		t.Fatalf("want 4 entries, got %d: %v", len(entries), entries)
	}
}

func TestParseList_PrefixFilter(t *testing.T) {
	entries := parseList(listOutput, "launchdude.")
	if len(entries) != 2 {
		t.Fatalf("want 2 launchdude.* entries, got %d: %v", len(entries), entries)
	}
	byLabel := map[string]ListEntry{}
	for _, e := range entries {
		byLabel[e.Label] = e
	}
	if byLabel["launchdude.alpha"].PID != 33333 {
		t.Errorf("alpha PID: want 33333, got %d", byLabel["launchdude.alpha"].PID)
	}
	if byLabel["launchdude.beta"].PID != 0 {
		t.Errorf("beta PID: want 0 (not running), got %d", byLabel["launchdude.beta"].PID)
	}
	if byLabel["launchdude.beta"].Status != 1 {
		t.Errorf("beta Status: want 1, got %d", byLabel["launchdude.beta"].Status)
	}
}

func TestFake_BootstrapKickstartStopBootout(t *testing.T) {
	fake := NewFake()
	const domain = "gui/100"
	const plistPath = "/tmp/fake/LaunchAgents/launchdude.example.plist"
	const target = "gui/100/launchdude.example"

	// Initial: not loaded.
	if _, err := fake.Print(target); err != ErrNotLoaded {
		t.Errorf("expected ErrNotLoaded, got %v", err)
	}

	// Bootstrap loads but doesn't start (RunAtLoad would normally trigger
	// kickstart in launchd; the fake leaves it waiting and the caller decides).
	if err := fake.Bootstrap(domain, plistPath); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	st, err := fake.Print(target)
	if err != nil {
		t.Fatalf("Print after bootstrap: %v", err)
	}
	if !st.Loaded {
		t.Error("Loaded should be true")
	}
	if st.PID != 0 {
		t.Errorf("PID before kickstart: want 0, got %d", st.PID)
	}

	// Second bootstrap is ErrAlreadyLoaded.
	if err := fake.Bootstrap(domain, plistPath); err != ErrAlreadyLoaded {
		t.Errorf("expected ErrAlreadyLoaded on re-bootstrap, got %v", err)
	}

	// Kickstart assigns a PID.
	if err := fake.Kickstart(target, false); err != nil {
		t.Fatal(err)
	}
	st, _ = fake.Print(target)
	if st.PID == 0 {
		t.Error("PID should be non-zero after kickstart")
	}
	firstPID := st.PID

	// Kickstart again without killExisting: PID unchanged (already running).
	if err := fake.Kickstart(target, false); err != nil {
		t.Fatal(err)
	}
	st, _ = fake.Print(target)
	if st.PID != firstPID {
		t.Errorf("Kickstart without -k should be no-op when running; PID %d -> %d", firstPID, st.PID)
	}

	// Stop and verify.
	if err := fake.Stop(target); err != nil {
		t.Fatal(err)
	}
	st, _ = fake.Print(target)
	if st.PID != 0 {
		t.Errorf("PID after stop: want 0, got %d", st.PID)
	}

	// Bootout removes it.
	if err := fake.Bootout(target); err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Print(target); err != ErrNotLoaded {
		t.Errorf("after bootout: want ErrNotLoaded, got %v", err)
	}
}
