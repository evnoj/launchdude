package cmd

import (
	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:               "status NAME",
	Short:             "Show the current state of a service",
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := manager.Status(args[0])
		if err != nil {
			return err
		}
		ui.PrintStatus(cmd.OutOrStdout(), st)
		return nil
	},
}
