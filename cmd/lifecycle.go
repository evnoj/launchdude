package cmd

import (
	"fmt"
	"os"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/service"
	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:               "apply NAME",
	Short:             "Re-render the plist from the TOML config and reload if loaded",
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.Apply(name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s applied\n", config.Label(name))
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit NAME",
	Short: "Open the service's TOML config in $EDITOR",
	Long: `Opens the existing config file in $VISUAL or $EDITOR (default vi).
Validates on save; if invalid, prompts to re-edit / dump / save-as / discard
(original file is preserved on discard). Renaming via edit is not allowed —
delete and recreate to rename. Run ` + "`launchdude apply NAME`" + ` afterward to reload.`,
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := config.ValidateName(name); err != nil {
			return err
		}
		path, err := config.ServiceConfigPath(name)
		if err != nil {
			return err
		}
		original, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("no config at %s; use `launchdude create %s` first", path, name)
		}
		if err != nil {
			return err
		}

		result, err := ui.EditAndValidate(original, func(data []byte) []error {
			svc, errs := config.ParseAndValidate(data)
			if svc == nil {
				return errs
			}
			if err := config.CheckNameMatches(svc, name); err != nil {
				errs = append(errs, err)
			}
			return errs
		})
		if err != nil {
			return err
		}
		switch result.Outcome {
		case ui.OutcomeDiscarded:
			fmt.Fprintln(cmd.OutOrStderr(), "discarded — original config unchanged")
			return nil
		case ui.OutcomeDumped, ui.OutcomeSavedExternal:
			fmt.Fprintln(cmd.OutOrStderr(), "original config unchanged")
			return nil
		case ui.OutcomeSuccess:
		}

		if err := os.WriteFile(path, result.Bytes, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "edited %s — run `launchdude apply %s` to reload\n", path, name)
		return nil
	},
}

var (
	deleteForce bool

	deleteCmd = &cobra.Command{
		Use:               "delete NAME",
		Short:             "Stop, deregister, and remove the service",
		Long:              "Removes the launchagent and the TOML config. Idempotent on each step. Use --force to skip the confirmation prompt.",
		Args:              requireArgs("NAME"),
		ValidArgsFunction: serviceNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.ValidateName(name); err != nil {
				return err
			}
			cfgPath, err := config.ServiceConfigPath(name)
			if err != nil {
				return err
			}
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				return fmt.Errorf("no service named %q (no config at %s)", name, cfgPath)
			} else if err != nil {
				return err
			}
			if !deleteForce {
				fmt.Fprintf(cmd.OutOrStderr(), "delete %s? this removes the launchagent and the config. [y/N] ", config.Label(name))
				var ans string
				_, _ = fmt.Fscanln(os.Stdin, &ans)
				if ans != "y" && ans != "Y" && ans != "yes" {
					return fmt.Errorf("aborted")
				}
			}
			if err := manager.Delete(name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s deleted\n", config.Label(name))
			return nil
		},
	}
)

var startCmd = &cobra.Command{
	Use:               "start NAME",
	Short:             "Start a loaded service (no-op if already running)",
	Long:              "Starts whatever plist is currently installed; does not re-render from TOML. Use `apply` or `restart` to pick up TOML changes.",
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.Start(name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s started\n", config.Label(name))
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:               "stop NAME",
	Short:             "Stop a running service (no-op if already stopped)",
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.Stop(name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s stopped\n", config.Label(name))
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:               "restart NAME",
	Short:             "Re-apply the config and restart the service with a fresh process",
	Args:              requireArgs("NAME"),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := manager.Restart(name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s restarted\n", config.Label(name))
		return nil
	},
}

var (
	disableNoStop bool

	disableCmd = &cobra.Command{
		Use:               "disable NAME",
		Short:             "Stop and deregister a service (TOML config preserved)",
		Long:              "Boots the service out of launchd. By default stops it first if running. Idempotent. Use --no-stop to skip the stop.",
		Args:              requireArgs("NAME"),
		ValidArgsFunction: serviceNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := manager.Disable(name, service.DisableOpts{NoStop: disableNoStop}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s disabled\n", config.Label(name))
			return nil
		},
	}
)

func init() {
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "skip the confirmation prompt")
	disableCmd.Flags().BoolVar(&disableNoStop, "no-stop", false, "deregister without stopping the running process first")
}
