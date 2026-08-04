package cmd

import (
	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:               "show NAME",
	Short:             "Show a service's state and configured properties",
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := manager.Status(args[0])
		if err != nil {
			return err
		}
		// Tolerate an unparseable/missing config the same way Status does:
		// still show the state block, just omit the property rows.
		svc, _ := config.LoadService(args[0])
		ui.PrintShow(cmd.OutOrStdout(), st, svc)
		return nil
	},
}
