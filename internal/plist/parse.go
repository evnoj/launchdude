package plist

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/evnoj/launchdude/internal/config"
	"howett.net/plist"
)

// parsePlist mirrors rawPlist but is permissive: it accepts plists not written
// by launchdude (no provenance keys, missing optional fields, etc).
// KeepAlive in launchd can be either a bool or a dict; we accept interface{}
// and best-effort flatten to a bool.
type parsePlist struct {
	Label                string            `plist:"Label"`
	Program              string            `plist:"Program,omitempty"`
	ProgramArguments     []string          `plist:"ProgramArguments,omitempty"`
	WorkingDirectory     string            `plist:"WorkingDirectory,omitempty"`
	RunAtLoad            bool              `plist:"RunAtLoad,omitempty"`
	KeepAlive            any               `plist:"KeepAlive,omitempty"`
	StandardOutPath      string            `plist:"StandardOutPath,omitempty"`
	StandardErrorPath    string            `plist:"StandardErrorPath,omitempty"`
	EnvironmentVariables map[string]string `plist:"EnvironmentVariables,omitempty"`
	LaunchdudeHash       string            `plist:"LaunchdudeHash,omitempty"`
}

// ParseHash extracts just the LaunchdudeHash key from a plist's bytes.
// Returns empty string if the plist has no such key (e.g., a plist not written
// by launchdude). Caller decides whether absence means "skip drift check" or
// "treat as drifted".
func ParseHash(data []byte) (string, error) {
	var p parsePlist
	dec := plist.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&p); err != nil {
		return "", err
	}
	return p.LaunchdudeHash, nil
}

// HashConfig computes a stable hash of a service's behavioral content. Two
// TOML files that parse to the same ServiceConfig produce the same hash, so
// comment-only / whitespace-only / env-key-reordering edits don't trigger
// drift.
//
// The hash is over the re-encoded canonical TOML of the parsed Service.
// BurntSushi/toml emits struct fields in declaration order and sorts map keys
// alphabetically, so the encoder output is deterministic for our schema.
// See TestTOMLEncoderDeterministic for a regression guard.
func HashConfig(svc *config.Service) string {
	var buf bytes.Buffer
	_ = toml.NewEncoder(&buf).Encode(svc)
	sum := sha256.Sum256(buf.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Parse reads an XML plist's bytes and returns the equivalent ServiceConfig
// plus the launchdude name (stripped of the launchdude. prefix if present).
// If the plist has no Label or the label doesn't carry the prefix, name is the
// raw label and the caller decides what to do with it.
func Parse(data []byte) (svc *config.Service, name string, err error) {
	var p parsePlist
	dec := plist.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&p); err != nil {
		return nil, "", fmt.Errorf("decode plist: %w", err)
	}
	if p.Label == "" {
		return nil, "", fmt.Errorf("plist has no Label key")
	}

	args := p.ProgramArguments
	if len(args) == 0 && p.Program != "" {
		args = []string{p.Program}
	}

	if n, ok := config.NameFromLabel(p.Label); ok {
		name = n
	} else {
		name = p.Label
	}

	svc = &config.Service{
		Name:       name,
		ExecArgs:   args,
		WorkingDir: p.WorkingDirectory,
		RunAtLoad:  p.RunAtLoad,
		KeepAlive:  coerceKeepAlive(p.KeepAlive),
		StdoutPath: p.StandardOutPath,
		StderrPath: p.StandardErrorPath,
		Env:        p.EnvironmentVariables,
	}
	return svc, name, nil
}

func coerceKeepAlive(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case nil:
		return false
	default:
		// Dict-form KeepAlive (e.g., {SuccessfulExit: false}) — treat as true
		// since the service is meant to be kept alive under some condition.
		return true
	}
}
