package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// requireArgs is the replacement for cobra.ExactArgs that names the missing
// positional arguments instead of saying "accepts 1 arg(s), received 0".
// Returns a *MissingArgsError when too few are given, which Execute()
// recognizes to also print the subcommand's help as a hint (when interactive).
func requireArgs(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, given []string) error {
		if len(given) < len(names) {
			return &MissingArgsError{Cmd: cmd, Missing: names[len(given):]}
		}
		if len(given) > len(names) {
			return &ExtraArgsError{Cmd: cmd, Expected: names, GotCount: len(given)}
		}
		return nil
	}
}

// MissingArgsError carries the offending command so the top-level error
// handler can print its help. Behaves as a plain error otherwise.
type MissingArgsError struct {
	Cmd     *cobra.Command
	Missing []string
}

func (e *MissingArgsError) Error() string {
	return "requires " + strings.Join(e.Missing, " ")
}

// ExtraArgsError is the symmetric case: too many positional args.
type ExtraArgsError struct {
	Cmd      *cobra.Command
	Expected []string
	GotCount int
}

func (e *ExtraArgsError) Error() string {
	if len(e.Expected) == 0 {
		return "this command takes no positional arguments"
	}
	return "too many arguments: this command takes " + strings.Join(e.Expected, " ")
}
