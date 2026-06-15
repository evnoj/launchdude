package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid service name %q: must match %s", name, nameRe.String())
	}
	return nil
}

type Service struct {
	Name        string            `toml:"name"`
	Exec        string            `toml:"exec,omitempty"`
	ExecArgs    []string          `toml:"exec_args,omitempty"`
	WorkingDir  string            `toml:"working_dir,omitempty"`
	KeepAlive   bool              `toml:"keep_alive"`
	RunAtLoad   bool              `toml:"run_at_load"`
	StdoutPath  string            `toml:"stdout_path,omitempty"`
	StderrPath  string            `toml:"stderr_path,omitempty"`
	Env         map[string]string `toml:"env,omitempty"`
}

// Validate returns ALL validation errors (not just the first) so the user can
// fix everything in one pass through the editor.
func (s *Service) Validate() []error {
	var errs []error
	if s.Name == "" {
		errs = append(errs, fmt.Errorf("name: required"))
	} else if err := ValidateName(s.Name); err != nil {
		errs = append(errs, fmt.Errorf("name: %w", err))
	}
	if s.Exec == "" && len(s.ExecArgs) == 0 {
		errs = append(errs, fmt.Errorf("exec: required (or specify exec_args)"))
	}
	if s.Exec != "" && len(s.ExecArgs) > 0 {
		errs = append(errs, fmt.Errorf("exec: cannot specify both exec and exec_args"))
	}
	return errs
}

func LoadService(name string) (*Service, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	path, err := ServiceConfigPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	svc, errs := ParseAndValidate(data)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid config at %s: %s", path, errsToString(errs))
	}
	if svc.Name != name {
		return nil, fmt.Errorf("config at %s declares name=%q which doesn't match the filename", path, svc.Name)
	}
	return svc, nil
}

func errsToString(errs []error) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "; ")
}

func SaveService(name string, svc *Service) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	dir, err := ServicesDir()
	if err != nil {
		return err
	}
	if err := EnsureDir(dir); err != nil {
		return err
	}
	path, err := ServiceConfigPath(name)
	if err != nil {
		return err
	}
	var sb strings.Builder
	if err := toml.NewEncoder(&sb).Encode(svc); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// TrashServiceConfig moves services/<name>.toml to the macOS Trash.
// Returns nil if the file is already gone (idempotent).
func TrashServiceConfig(name string) error {
	path, err := ServiceConfigPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return TrashFile(path)
}

// ParseAndValidate parses raw TOML bytes into a Service and runs the schema
// validator. Strict: any TOML keys not in the Service struct are reported as
// errors (typo'd or removed fields fail loudly rather than being silently
// ignored).
//
// Returns:
//   - svc: nil if the TOML failed to parse, otherwise the parsed struct
//     (even if it failed schema validation, so callers can show what was read).
//   - errs: all errors encountered, schema and parse alike. Empty on success.
func ParseAndValidate(data []byte) (*Service, []error) {
	var svc Service
	md, err := toml.Decode(string(data), &svc)
	if err != nil {
		return nil, []error{fmt.Errorf("toml: %w", err)}
	}
	var errs []error
	for _, k := range md.Undecoded() {
		errs = append(errs, fmt.Errorf("unknown TOML key: %s", k))
	}
	errs = append(errs, svc.Validate()...)
	return &svc, errs
}

// CheckNameAvailable returns an error if a service config already exists at
// services/<name>.toml. Used by `create` to reject saving over an existing service.
func CheckNameAvailable(name string) error {
	path, err := ServiceConfigPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("name %q: service already exists (edit with `launchdude edit %s` or pick a different name)", name, name)
	}
	return nil
}

// CheckNameMatches returns an error if the parsed service's Name doesn't match
// the expected name. Used by `edit` to forbid renaming via edit
// (force users to delete + create for a rename).
func CheckNameMatches(svc *Service, expected string) error {
	if svc.Name != expected {
		return fmt.Errorf("name %q: cannot change in `edit` (got %q, expected %q); delete and recreate to rename", svc.Name, svc.Name, expected)
	}
	return nil
}

func ListServiceNames() ([]string, error) {
	dir, err := ServicesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".toml") {
			continue
		}
		base := strings.TrimSuffix(n, ".toml")
		if ValidateName(base) == nil {
			names = append(names, base)
		}
	}
	sort.Strings(names)
	return names, nil
}

// ResolveStdoutPath returns the effective stdout path: explicit override if set,
// else the auto-routed default.
func (s *Service) ResolveStdoutPath(name string) (string, error) {
	if s.StdoutPath != "" {
		return expandUser(s.StdoutPath)
	}
	return DefaultStdoutPath(name)
}

func (s *Service) ResolveStderrPath(name string) (string, error) {
	if s.StderrPath != "" {
		return expandUser(s.StderrPath)
	}
	return DefaultStderrPath(name)
}

func (s *Service) ResolveWorkingDir() (string, error) {
	if s.WorkingDir == "" {
		return "", nil
	}
	return expandUser(s.WorkingDir)
}

func expandUser(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
