package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EditOutcome describes how an EditAndValidate session ended.
type EditOutcome int

const (
	// OutcomeSuccess: user saved bytes that passed validation.
	OutcomeSuccess EditOutcome = iota
	// OutcomeDiscarded: user gave up; no bytes were preserved.
	OutcomeDiscarded
	// OutcomeDumped: user printed their work to the terminal and exited.
	OutcomeDumped
	// OutcomeSavedExternal: user wrote their work to a path of their choosing,
	// outside launchdude's managed locations.
	OutcomeSavedExternal
)

// EditResult is returned from EditAndValidate. Only Bytes is populated for
// OutcomeSuccess; the other outcomes mean the caller should not install
// anything (the user's content has either been printed, saved elsewhere, or
// thrown away).
type EditResult struct {
	Outcome EditOutcome
	Bytes   []byte
}

// EditAndValidate writes `initial` to a temp file, opens it in $EDITOR, and
// after the editor closes runs `validate` on the saved bytes. On validation
// failure, prompts the user with e/d/s/q recovery options. Loops on `e` so
// the user can iterate without losing work.
//
// The temp file is cleaned up before return.
func EditAndValidate(initial []byte, validate func([]byte) []error) (*EditResult, error) {
	tmp, err := os.CreateTemp("", "launchdude-*.toml")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(initial); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}

	r := bufio.NewReader(os.Stdin)

	for {
		if err := openEditor(tmpPath); err != nil {
			return nil, fmt.Errorf("editor: %w", err)
		}
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return nil, err
		}
		errs := validate(data)
		if len(errs) == 0 {
			return &EditResult{Outcome: OutcomeSuccess, Bytes: data}, nil
		}

		fmt.Fprintln(os.Stderr, Failed.Render("config invalid:"))
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		c := Prompt(r, os.Stderr, "\n  e)dit again  d)ump to terminal  s)ave to a path  q)uit > ")
		switch c {
		case 'e', 0:
			continue
		case 'd':
			fmt.Println("---")
			os.Stdout.Write(data)
			fmt.Println("---")
			return &EditResult{Outcome: OutcomeDumped}, nil
		case 's':
			fmt.Fprint(os.Stderr, "  path: ")
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "" {
				fmt.Fprintln(os.Stderr, "  no path given; back to menu")
				continue
			}
			line = expandUser(line)
			if err := os.WriteFile(line, data, 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "  error writing %s: %v\n", line, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "  wrote to %s\n", line)
			return &EditResult{Outcome: OutcomeSavedExternal}, nil
		case 'q':
			return &EditResult{Outcome: OutcomeDiscarded}, nil
		default:
			fmt.Fprintln(os.Stderr, "  invalid choice")
		}
	}
}

func openEditor(path string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command("sh", "-c", fmt.Sprintf("%s %q", editor, path))
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func expandUser(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:]
		}
	}
	return p
}
