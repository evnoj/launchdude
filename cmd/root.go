package cmd

import (
	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/launchctl"
	"github.com/evnoj/launchdude/internal/service"
	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var (
	noColor bool

	// manager is shared by all subcommands. Lazily initialized so tests can
	// inject a fake by setting this directly before Execute().
	manager *service.Manager
)

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "launchdude",
	Short: "Manage macOS launchd services",
	Long: `launchdude is a friendly interface to launchctl for managing user-level
launch agents (~/Library/LaunchAgents). Service configs live as TOML in
$XDG_CONFIG_HOME/launchdude/services/ and are rendered to plists on demand.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor {
			ui.DisableColor()
		}
		if manager == nil {
			manager = service.New(launchctl.New())
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.AddCommand(
		createCmd, enableCmd, statusCmd,
		applyCmd, editCmd, deleteCmd, importCmd,
		startCmd, stopCmd, restartCmd, disableCmd,
		listCmd, logsCmd, doctorCmd,
	)
}

// serviceNameCompletion provides shell completion for service-name positional args.
func serviceNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	names, err := config.ListServiceNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
