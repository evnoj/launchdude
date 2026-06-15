package ui

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
)

// Interactive is true when stdout is a real terminal. Set once at startup
// (an open file's Fd never changes). LAUNCHDUDE_INTERACTIVE=1 forces it on
// for testing or for piping into a pager you still want hints in.
var Interactive = isatty.IsTerminal(os.Stdout.Fd()) || os.Getenv("LAUNCHDUDE_INTERACTIVE") == "1"

// Hint writes a helpful tip to stdout, but only when stdout is a TTY.
// Unstyled, unprefixed — caller writes plain English.
//
// Use for "run X next" pointers and similar tips that would clutter scripted
// output. Anything essential (results, errors) should NOT be a hint.
func Hint(format string, a ...any) {
	if !Interactive {
		return
	}
	fmt.Fprintln(os.Stdout, fmt.Sprintf(format, a...))
}
