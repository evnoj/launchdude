package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var (
	createNoEdit   bool
	createTemplate string
)

var createCmd = &cobra.Command{
	Use:   "create [NAME]",
	Short: "Create a new service config and open it in $EDITOR",
	Long: `Opens $EDITOR with a TOML template. If NAME is provided, it pre-populates
the name field. The service is saved at $XDG_CONFIG_HOME/launchdude/services/<NAME>.toml
on successful save (where NAME comes from the saved TOML, not the CLI arg).

On invalid save: prompts to re-edit, dump to terminal, save to a path, or discard.
Does not install or load the service — use ` + "`launchdude enable NAME`" + ` after.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		seedName := ""
		if len(args) == 1 {
			seedName = args[0]
			if err := config.ValidateName(seedName); err != nil {
				return err
			}
		}

		var tmpl string
		if createTemplate != "" {
			data, err := os.ReadFile(createTemplate)
			if err != nil {
				return fmt.Errorf("read template %s: %w", createTemplate, err)
			}
			tmpl = string(data)
		} else {
			t, err := config.Template(seedName)
			if err != nil {
				return err
			}
			tmpl = t
		}

		if createNoEdit {
			// No-edit mode: requires NAME and writes the template directly.
			// Skips the validate-and-recover flow because there's no editor.
			if seedName == "" {
				return fmt.Errorf("--no-edit requires a NAME positional argument")
			}
			if err := config.CheckNameAvailable(seedName); err != nil {
				return err
			}
			path, err := config.ServiceConfigPath(seedName)
			if err != nil {
				return err
			}
			if err := config.EnsureDir(filepath.Dir(path)); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(tmpl), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", ui.Running.Render("created"), path)
			ui.Hint("service is not running yet; run `launchdude enable %s` to register and start it", seedName)
			return nil
		}

		result, err := ui.EditAndValidate([]byte(tmpl), func(data []byte) []error {
			svc, errs := config.ParseAndValidate(data)
			if svc == nil {
				return errs
			}
			if err := config.CheckNameAvailable(svc.Name); err != nil {
				errs = append(errs, err)
			}
			return errs
		})
		if err != nil {
			return err
		}
		switch result.Outcome {
		case ui.OutcomeDiscarded:
			fmt.Fprintln(cmd.OutOrStderr(), "discarded")
			return nil
		case ui.OutcomeDumped, ui.OutcomeSavedExternal:
			// User already saw / received their work; nothing more to do.
			return nil
		case ui.OutcomeSuccess:
		}

		svc, _ := config.ParseAndValidate(result.Bytes) // validated above
		dest, err := config.ServiceConfigPath(svc.Name)
		if err != nil {
			return err
		}
		if err := config.EnsureDir(filepath.Dir(dest)); err != nil {
			return err
		}
		if err := os.WriteFile(dest, result.Bytes, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", ui.Running.Render("created"), dest)
		ui.Hint("service is not running yet; run `launchdude enable %s` to register and start it", svc.Name)
		return nil
	},
}

func init() {
	createCmd.Flags().BoolVar(&createNoEdit, "no-edit", false, "don't open $EDITOR; requires NAME and skips validation")
	createCmd.Flags().StringVar(&createTemplate, "template", "", "path to a TOML template to use as starting content")
}
