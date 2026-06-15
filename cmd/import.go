package cmd

import (
	"fmt"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/service"
	"github.com/spf13/cobra"
)

var importForce bool

var importCmd = &cobra.Command{
	Use:   "import NAME",
	Short: "Adopt an existing launchdude.NAME.plist into a TOML config",
	Long: `Reads ~/Library/LaunchAgents/launchdude.NAME.plist and writes the equivalent
TOML to $XDG_CONFIG_HOME/launchdude/services/NAME.toml. Useful for recovering
from a deleted TOML or adopting a plist created outside launchdude (as long as
it follows the launchdude.* label convention).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.Import(name, service.ImportOpts{Force: importForce}); err != nil {
			return err
		}
		path, _ := config.ServiceConfigPath(name)
		fmt.Fprintf(cmd.OutOrStdout(), "imported %s -> %s\n", config.Label(name), path)
		return nil
	},
}

func init() {
	importCmd.Flags().BoolVar(&importForce, "force", false, "overwrite an existing TOML config")
}
