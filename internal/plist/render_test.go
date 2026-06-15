package plist

import (
	"bytes"
	"strings"
	"testing"

	"github.com/evnoj/launchdude/internal/config"
	"howett.net/plist"
)

// rawAccess parses the rendered bytes into a generic map so tests can assert
// on individual plist keys without depending on Parse() (which is the unit
// under test for round-trip).
func rawAccess(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	dec := plist.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("decode rendered plist: %v", err)
	}
	return m
}

func TestRender_BasicFieldsAndLabel(t *testing.T) {
	svc := &config.Service{
		Name:      "myapi",
		Exec:      "/usr/local/bin/myapi --port 8080",
		KeepAlive: true,
		RunAtLoad: true,
	}
	data, err := Render("myapi", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	if got := m["Label"]; got != "launchdude.myapi" {
		t.Errorf("Label: want launchdude.myapi, got %v", got)
	}
	if got := m["KeepAlive"]; got != true {
		t.Errorf("KeepAlive: want true, got %v", got)
	}
	if got := m["RunAtLoad"]; got != true {
		t.Errorf("RunAtLoad: want true, got %v", got)
	}
}

func TestRender_ExecShellSplit(t *testing.T) {
	// `exec` should be shellwords-parsed into ProgramArguments.
	svc := &config.Service{
		Name: "x",
		Exec: `/bin/foo --port 8080 --name "with spaces"`,
	}
	data, err := Render("x", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	args, ok := m["ProgramArguments"].([]any)
	if !ok {
		t.Fatalf("ProgramArguments missing or wrong type: %T", m["ProgramArguments"])
	}
	want := []string{"/bin/foo", "--port", "8080", "--name", "with spaces"}
	if len(args) != len(want) {
		t.Fatalf("argc: want %d got %d (%v)", len(want), len(args), args)
	}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("arg %d: want %q got %v", i, w, args[i])
		}
	}
}

func TestRender_ExecArgsNotSplit(t *testing.T) {
	// exec_args bypasses shellwords; each element passes through verbatim.
	svc := &config.Service{
		Name:     "x",
		ExecArgs: []string{"/bin/foo", "--complicated arg with spaces", "and quotes \"in it\""},
	}
	data, err := Render("x", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	args := m["ProgramArguments"].([]any)
	if args[1] != svc.ExecArgs[1] {
		t.Errorf("exec_args[1] mangled: want %q got %v", svc.ExecArgs[1], args[1])
	}
	if args[2] != svc.ExecArgs[2] {
		t.Errorf("exec_args[2] mangled: want %q got %v", svc.ExecArgs[2], args[2])
	}
}

func TestRender_EnvMap(t *testing.T) {
	svc := &config.Service{
		Name: "x",
		Exec: "/bin/x",
		Env:  map[string]string{"FOO": "1", "BAR": "two"},
	}
	data, err := Render("x", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	env, ok := m["EnvironmentVariables"].(map[string]any)
	if !ok {
		t.Fatalf("EnvironmentVariables missing or wrong type: %T", m["EnvironmentVariables"])
	}
	if env["FOO"] != "1" || env["BAR"] != "two" {
		t.Errorf("env values wrong: %v", env)
	}
}

func TestRender_DefaultLogPaths(t *testing.T) {
	svc := &config.Service{Name: "myname", Exec: "/bin/x"}
	data, err := Render("myname", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	// Default paths embed the service name.
	stdout := m["StandardOutPath"].(string)
	stderr := m["StandardErrorPath"].(string)
	if !strings.HasSuffix(stdout, "myname.out.log") {
		t.Errorf("StandardOutPath: %s", stdout)
	}
	if !strings.HasSuffix(stderr, "myname.err.log") {
		t.Errorf("StandardErrorPath: %s", stderr)
	}
}

func TestRender_CustomLogPathsHonored(t *testing.T) {
	svc := &config.Service{
		Name:       "x",
		Exec:       "/bin/x",
		StdoutPath: "/var/log/custom.out",
		StderrPath: "/var/log/custom.err",
	}
	data, err := Render("x", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	if got := m["StandardOutPath"]; got != "/var/log/custom.out" {
		t.Errorf("StandardOutPath override: got %v", got)
	}
	if got := m["StandardErrorPath"]; got != "/var/log/custom.err" {
		t.Errorf("StandardErrorPath override: got %v", got)
	}
}

func TestRender_EmbedsProvenance(t *testing.T) {
	svc := &config.Service{Name: "x", Exec: "/bin/x"}
	data, err := Render("x", svc)
	if err != nil {
		t.Fatal(err)
	}
	m := rawAccess(t, data)
	if got, _ := m["LaunchdudeSource"].(string); got != "x.toml" {
		t.Errorf("LaunchdudeSource: got %v", got)
	}
	if v, _ := m["LaunchdudeVersion"].(string); v == "" {
		t.Error("LaunchdudeVersion should not be empty")
	}
	h, _ := m["LaunchdudeHash"].(string)
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("LaunchdudeHash: want sha256: prefix, got %q", h)
	}
}

func TestRender_ErrorsOnEmptyCommand(t *testing.T) {
	svc := &config.Service{Name: "x"}
	if _, err := Render("x", svc); err == nil {
		t.Fatal("expected error when neither exec nor exec_args is set")
	}
}

// TestRoundTrip verifies that Render → Parse produces a Service equivalent to
// the original for the fields launchd actually cares about. Specifically:
//   - Label round-trips to Name
//   - exec is materialized as ExecArgs after parse (since Parse can't tell
//     whether the user wrote `exec` or `exec_args` originally)
//   - env, working_dir, keep_alive, run_at_load round-trip exactly
func TestRoundTrip(t *testing.T) {
	original := &config.Service{
		Name:       "rt",
		ExecArgs:   []string{"/bin/foo", "--x", "y"},
		WorkingDir: "/Users/foo/code",
		KeepAlive:  true,
		RunAtLoad:  false,
		StdoutPath: "/tmp/rt.out",
		StderrPath: "/tmp/rt.err",
		Env:        map[string]string{"A": "1", "B": "2"},
	}
	data, err := Render("rt", original)
	if err != nil {
		t.Fatal(err)
	}
	parsed, name, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if name != "rt" {
		t.Errorf("name: want rt got %q", name)
	}
	if parsed.WorkingDir != original.WorkingDir {
		t.Errorf("WorkingDir mismatch")
	}
	if parsed.KeepAlive != original.KeepAlive {
		t.Errorf("KeepAlive mismatch")
	}
	if parsed.RunAtLoad != original.RunAtLoad {
		t.Errorf("RunAtLoad mismatch")
	}
	if parsed.StdoutPath != original.StdoutPath {
		t.Errorf("StdoutPath mismatch")
	}
	if parsed.StderrPath != original.StderrPath {
		t.Errorf("StderrPath mismatch")
	}
	if len(parsed.ExecArgs) != len(original.ExecArgs) {
		t.Fatalf("ExecArgs length mismatch: %v vs %v", parsed.ExecArgs, original.ExecArgs)
	}
	for i, a := range original.ExecArgs {
		if parsed.ExecArgs[i] != a {
			t.Errorf("ExecArgs[%d]: want %q got %q", i, a, parsed.ExecArgs[i])
		}
	}
	if parsed.Env["A"] != "1" || parsed.Env["B"] != "2" {
		t.Errorf("Env mismatch: %v", parsed.Env)
	}
}

func TestParseHash_PresentAndAbsent(t *testing.T) {
	// Plist rendered by launchdude has a hash.
	svc := &config.Service{Name: "x", Exec: "/bin/x"}
	data, _ := Render("x", svc)
	h, err := ParseHash(data)
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Error("hash should be present for launchdude-rendered plist")
	}

	// Hand-crafted plist with no LaunchdudeHash: ParseHash returns empty, no error.
	noHash := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>com.example.bare</string>
<key>ProgramArguments</key><array><string>/bin/echo</string></array>
</dict></plist>`)
	h, err = ParseHash(noHash)
	if err != nil {
		t.Fatal(err)
	}
	if h != "" {
		t.Errorf("hash should be empty for hand-crafted plist, got %q", h)
	}
}
