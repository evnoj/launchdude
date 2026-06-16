package cmd

import (
	"fmt"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/service"
	"github.com/spf13/cobra"
)

var enableNoStart bool

var enableCmd = &cobra.Command{
	Use:   "enable NAME",
	Short: "Register the service with launchd and start it",
	Long: `enable applies any pending TOML changes, registers the service with launchd
(if not already registered), and starts it (unless --no-start).

Idempotent — running enable on an already-enabled service is a no-op. This is
the behavior serviceman gets wrong by leaking "already loaded" errors.`,
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.Enable(name, service.EnableOpts{NoStart: enableNoStart}); err != nil {
			return err
		}
		label := config.Label(name)
		if enableNoStart {
			fmt.Fprintf(cmd.OutOrStdout(), "%s registered (start with `launchdude start %s`)\n", label, name)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%s enabled\n", label)
		}
		return nil
	},
}

func init() {
	enableCmd.Flags().BoolVar(&enableNoStart, "no-start", false, "register the service but don't start it")
}
