package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateHome points HOME and XDG_CONFIG_HOME at a per-test temp dir so the
// path helpers resolve to throwaway locations. Returns the home dir for
// further setup.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	return home
}

func TestValidateName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"foo", true},
		{"foo-bar", true},
		{"foo.bar", true},
		{"foo_bar", true},
		{"foo123", true},
		{"123foo", true}, // leading digit is fine, regex starts with [a-zA-Z0-9]
		{"", false},
		{"-foo", false},   // leading dash
		{"foo bar", false},
		{"foo/bar", false},
		{".foo", false},   // leading dot
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateName(c.name)
			if c.ok && err != nil {
				t.Errorf("expected ok, got %v", err)
			}
			if !c.ok && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestServiceValidateReportsAllErrors(t *testing.T) {
	// Empty service is missing both required fields. Validate must return both,
	// not just the first — that's the whole point of the []error contract so
	// the editor recovery prompt can show every problem at once.
	s := &Service{}
	errs := s.Validate()
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors (name + exec), got %d: %v", len(errs), errs)
	}
	joined := strings.Join(errsToStrings(errs), "; ")
	if !strings.Contains(joined, "name") {
		t.Errorf("expected name error in %q", joined)
	}
	if !strings.Contains(joined, "exec") {
		t.Errorf("expected exec error in %q", joined)
	}
}

func TestServiceValidateBothExecAndExecArgs(t *testing.T) {
	s := &Service{Name: "x", Exec: "/bin/x", ExecArgs: []string{"/bin/x"}}
	errs := s.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for both exec and exec_args set")
	}
}

func TestServiceValidateNamePassesRegex(t *testing.T) {
	s := &Service{Name: "bad name", Exec: "/bin/x"}
	errs := s.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for invalid name pattern")
	}
}

func TestParseAndValidateStrict_TypoIsError(t *testing.T) {
	src := []byte(`name = "foo"
exec = "/bin/x"
descriptoin = "typo"
`)
	_, errs := ParseAndValidate(src)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown key")
	}
	joined := strings.Join(errsToStrings(errs), "; ")
	if !strings.Contains(joined, "descriptoin") {
		t.Errorf("expected error mentioning 'descriptoin', got %q", joined)
	}
}

func TestParseAndValidateValid(t *testing.T) {
	src := []byte(`name = "foo"
exec = "/bin/x"
keep_alive = true
run_at_load = false
[env]
A = "1"
`)
	svc, errs := ParseAndValidate(src)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if svc.Name != "foo" {
		t.Errorf("name: want foo got %q", svc.Name)
	}
	if svc.Exec != "/bin/x" {
		t.Errorf("exec: want /bin/x got %q", svc.Exec)
	}
	if !svc.KeepAlive {
		t.Error("keep_alive should be true")
	}
	if svc.RunAtLoad {
		t.Error("run_at_load should be false")
	}
	if svc.Env["A"] != "1" {
		t.Errorf("env A: want 1 got %q", svc.Env["A"])
	}
}

func TestParseAndValidate_TOMLParseError(t *testing.T) {
	// Unterminated string is an unambiguous TOML syntax error.
	src := []byte(`name = "unterminated`)
	svc, errs := ParseAndValidate(src)
	if svc != nil {
		t.Error("svc should be nil on parse error")
	}
	if len(errs) == 0 {
		t.Fatal("expected parse error")
	}
}

func TestLoadServiceRejectsNameMismatch(t *testing.T) {
	isolateHome(t)
	dir, _ := ServicesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, _ := ServiceConfigPath("foo")
	body := []byte(`name = "bar"
exec = "/bin/x"
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadService("foo")
	if err == nil {
		t.Fatal("expected error: name doesn't match filename")
	}
	if !strings.Contains(err.Error(), "doesn't match") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckNameAvailable(t *testing.T) {
	isolateHome(t)

	if err := CheckNameAvailable("nonexistent"); err != nil {
		t.Errorf("nonexistent should be available, got %v", err)
	}

	// Create the file and re-check.
	dir, _ := ServicesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, _ := ServiceConfigPath("exists")
	if err := os.WriteFile(path, []byte(`name = "exists"`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckNameAvailable("exists"); err == nil {
		t.Error("expected conflict error for existing service")
	}
}

func TestCheckNameMatches(t *testing.T) {
	if err := CheckNameMatches(&Service{Name: "foo"}, "foo"); err != nil {
		t.Errorf("matching name: %v", err)
	}
	if err := CheckNameMatches(&Service{Name: "bar"}, "foo"); err == nil {
		t.Error("expected mismatch error")
	}
}

func TestNameFromLabel(t *testing.T) {
	if n, ok := NameFromLabel("launchdude.foo"); !ok || n != "foo" {
		t.Errorf("launchdude.foo: got (%q, %v)", n, ok)
	}
	if _, ok := NameFromLabel("com.apple.foo"); ok {
		t.Error("non-launchdude label should not match")
	}
	if _, ok := NameFromLabel("launchdude."); ok {
		t.Error("prefix-only label should not match (no name)")
	}
}

func errsToStrings(errs []error) []string {
	out := make([]string, len(errs))
	for i, e := range errs {
		out[i] = e.Error()
	}
	return out
}
