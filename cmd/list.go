package cmd

import (
	"fmt"

	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var listAll bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all launchdude services and their state",
	Long:    "Shows every service known via its TOML config, installed plist, or current launchd registration. Use --all to also dump every other user-domain launchagent (read-only — launchdude can't manage non-launchdude services).",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := manager.List()
		if err != nil {
			return err
		}
		ui.PrintList(cmd.OutOrStdout(), entries)
		if !listAll {
			return nil
		}
		labels, err := manager.AllUserAgents()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), ui.Bold.Render("OTHER USER AGENTS (read-only)"))
		for _, l := range labels {
			if l == "" {
				continue
			}
			fmt.Fprintln(cmd.OutOrStdout(), "  "+l)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "also list non-launchdude user agents (read-only)")
}
