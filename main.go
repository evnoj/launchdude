package main

import (
	"fmt"
	"os"

	"github.com/evnoj/launchdude/cmd"
	"github.com/evnoj/launchdude/internal/ui"
)

func main() {
	err := cmd.Execute()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "launchdude:", err)
	// When a required positional arg was missing AND we're interactive, follow
	// the terse error with the subcommand's help — mirrors `ui.Hint`'s
	// "interactive-only guidance" pattern at the top-level error surface.
	if mae, ok := err.(*cmd.MissingArgsError); ok && ui.Interactive {
		fmt.Fprintln(os.Stderr)
		_ = mae.Cmd.Help()
	}
	os.Exit(1)
}
