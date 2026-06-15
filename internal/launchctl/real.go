package launchctl

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Real shells out to the system `launchctl` binary.
type Real struct{}

func New() *Real { return &Real{} }

func (r *Real) Bootstrap(domain, plistPath string) error {
	_, stderr, err := run("launchctl", "bootstrap", domain, plistPath)
	if err == nil {
		return nil
	}
	if isAlreadyLoaded(stderr) {
		return ErrAlreadyLoaded
	}
	if isDisabled(stderr) {
		return ErrDisabled
	}
	return fmt.Errorf("launchctl bootstrap: %w (%s)", err, condense(stderr))
}

func (r *Real) Bootout(serviceTarget string) error {
	_, stderr, err := run("launchctl", "bootout", serviceTarget)
	if err == nil {
		return nil
	}
	if isNotLoaded(stderr) {
		return ErrNotLoaded
	}
	return fmt.Errorf("launchctl bootout: %w (%s)", err, condense(stderr))
}

func (r *Real) Kickstart(serviceTarget string, killExisting bool) error {
	args := []string{"kickstart"}
	if killExisting {
		args = append(args, "-k")
	}
	args = append(args, serviceTarget)
	_, stderr, err := run("launchctl", args...)
	if err == nil {
		return nil
	}
	if isNotLoaded(stderr) {
		return ErrNotLoaded
	}
	return fmt.Errorf("launchctl kickstart: %w (%s)", err, condense(stderr))
}

func (r *Real) Stop(serviceTarget string) error {
	_, stderr, err := run("launchctl", "kill", "SIGTERM", serviceTarget)
	if err == nil {
		return nil
	}
	if isNotLoaded(stderr) {
		return ErrNotLoaded
	}
	// `launchctl kill` errors if the service isn't running, but that's the
	// state we want — treat as success.
	if isNotRunning(stderr) {
		return nil
	}
	return fmt.Errorf("launchctl stop: %w (%s)", err, condense(stderr))
}

func (r *Real) Print(serviceTarget string) (*ServiceState, error) {
	stdout, stderr, err := run("launchctl", "print", serviceTarget)
	if err != nil {
		if isNotLoaded(stderr) {
			return nil, ErrNotLoaded
		}
		return nil, fmt.Errorf("launchctl print: %w (%s)", err, condense(stderr))
	}
	return parsePrint(stdout), nil
}

func (r *Real) List(prefix string) ([]ListEntry, error) {
	stdout, stderr, err := run("launchctl", "list")
	if err != nil {
		return nil, fmt.Errorf("launchctl list: %w (%s)", err, condense(stderr))
	}
	return parseList(stdout, prefix), nil
}

func run(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// stderr classification — defensive. macOS surfaces "already loaded" with
// different errnos (5/I/O error, 17/file exists, 37) across versions, so we
// match on stable substrings rather than exit codes.
func isAlreadyLoaded(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "service already loaded") ||
		strings.Contains(s, "already loaded") ||
		strings.Contains(s, "input/output error") // observed in user's serviceman report
}

func isNotLoaded(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "could not find service") ||
		strings.Contains(s, "no such process") ||
		strings.Contains(s, "could not find specified service")
}

func isNotRunning(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "service is not running")
}

func isDisabled(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "service is disabled")
}

// condense flattens multiline stderr to a single line for error wrapping.
func condense(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "; ")
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
