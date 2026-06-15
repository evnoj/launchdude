package plist

import (
	"bytes"
	"fmt"

	"github.com/evnoj/launchdude/internal/config"
	shellwords "github.com/mattn/go-shellwords"
	"howett.net/plist"
)

// Version is overridable from main via ldflags later; for now hardcoded.
var Version = "0.1.0"

// rawPlist is the in-memory shape of the launchd plist we emit.
// Field order here doesn't matter for launchd, but the plist encoder
// emits XML keys in declaration order, which makes diffs friendlier.
type rawPlist struct {
	Label                string            `plist:"Label"`
	ProgramArguments     []string          `plist:"ProgramArguments"`
	WorkingDirectory     string            `plist:"WorkingDirectory,omitempty"`
	RunAtLoad            bool              `plist:"RunAtLoad"`
	KeepAlive            bool              `plist:"KeepAlive"`
	StandardOutPath      string            `plist:"StandardOutPath"`
	StandardErrorPath    string            `plist:"StandardErrorPath"`
	EnvironmentVariables map[string]string `plist:"EnvironmentVariables,omitempty"`

	// Provenance — written by launchdude, used for drift detection and recovery.
	LaunchdudeVersion string `plist:"LaunchdudeVersion"`
	LaunchdudeSource  string `plist:"LaunchdudeSource"`
	LaunchdudeHash    string `plist:"LaunchdudeHash"`
}

// Render produces the XML plist bytes for a service. The embedded
// LaunchdudeHash is computed from the parsed Service (not the raw TOML), so
// cosmetic edits to the TOML don't change the rendered plist.
func Render(name string, svc *config.Service) ([]byte, error) {
	args, err := resolveArgs(svc)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("service %q has no executable command", name)
	}

	workDir, err := svc.ResolveWorkingDir()
	if err != nil {
		return nil, err
	}
	stdout, err := svc.ResolveStdoutPath(name)
	if err != nil {
		return nil, err
	}
	stderr, err := svc.ResolveStderrPath(name)
	if err != nil {
		return nil, err
	}

	p := rawPlist{
		Label:                config.Label(name),
		ProgramArguments:     args,
		WorkingDirectory:     workDir,
		RunAtLoad:            svc.RunAtLoad,
		KeepAlive:            svc.KeepAlive,
		StandardOutPath:      stdout,
		StandardErrorPath:    stderr,
		EnvironmentVariables: svc.Env,
		LaunchdudeVersion:    Version,
		LaunchdudeSource:     name + ".toml",
	}
	p.LaunchdudeHash = HashConfig(svc)

	var buf bytes.Buffer
	enc := plist.NewEncoder(&buf)
	enc.Indent("\t")
	if err := enc.Encode(p); err != nil {
		return nil, fmt.Errorf("encode plist: %w", err)
	}
	return buf.Bytes(), nil
}

func resolveArgs(svc *config.Service) ([]string, error) {
	if len(svc.ExecArgs) > 0 {
		return svc.ExecArgs, nil
	}
	if svc.Exec == "" {
		return nil, nil
	}
	parser := shellwords.NewParser()
	parser.ParseEnv = false
	parser.ParseBacktick = false
	args, err := parser.Parse(svc.Exec)
	if err != nil {
		return nil, fmt.Errorf("parse exec %q: %w", svc.Exec, err)
	}
	return args, nil
}
