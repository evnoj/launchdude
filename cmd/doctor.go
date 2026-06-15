package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/evnoj/launchdude/internal/service"
	"github.com/evnoj/launchdude/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Interactively reconcile drift between TOML configs and launchagents",
	Long: `Scans for two kinds of discrepancy:
  - orphan:   plist exists at ~/Library/LaunchAgents/launchdude.NAME.plist
              but no TOML at $XDG_CONFIG_HOME/launchdude/services/NAME.toml
  - modified: TOML mtime is newer than the rendered plist mtime

Pending state (TOML exists, no plist) is NOT a discrepancy — it's an unfinished
install. Use ` + "`launchdude enable NAME`" + ` to complete it.

Skipped entries don't persist; they surface again next run.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		orphans, err := service.OrphanedPlists()
		if err != nil {
			return err
		}
		modified, err := modifiedServices()
		if err != nil {
			return err
		}
		total := len(orphans) + len(modified)
		if total == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), ui.Dim.Render("no discrepancies"))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%d discrepancy(ies) to review\n\n", total)
		r := bufio.NewReader(os.Stdin)
		idx := 0

		for _, name := range orphans {
			idx++
			if quit := handleOrphan(cmd.OutOrStdout(), r, idx, total, name); quit {
				return nil
			}
		}
		for _, name := range modified {
			idx++
			if quit := handleModified(cmd.OutOrStdout(), r, idx, total, name); quit {
				return nil
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), ui.Dim.Render("doctor done"))
		return nil
	},
}

// modifiedServices returns names where TOML mtime > plist mtime.
func modifiedServices() ([]string, error) {
	names, err := config.ListServiceNames()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range names {
		st, err := manager.Status(n)
		if err != nil {
			continue
		}
		if st.Drifted {
			out = append(out, n)
		}
	}
	return out, nil
}

func handleOrphan(w io.Writer, r *bufio.Reader, idx, total int, name string) bool {
	label := config.Label(name)
	st, _ := manager.Status(name)
	plistPath, _ := config.PlistPath(name)
	cfgPath, _ := config.ServiceConfigPath(name)

	fmt.Fprintf(w, "[%d/%d] %s: %s\n", idx, total, ui.Bold.Render(label), ui.Orphan.Render("orphan"))
	fmt.Fprintf(w, "  plist:    %s\n", plistPath)
	fmt.Fprintf(w, "  missing:  %s\n", cfgPath)
	if st != nil {
		if st.PID > 0 {
			fmt.Fprintf(w, "  running:  %s\n", ui.Running.Render(fmt.Sprintf("pid %d", st.PID)))
		} else if st.Loaded {
			fmt.Fprintf(w, "  loaded:   %s\n", ui.Stopped.Render("yes, not running"))
		}
	}
	for {
		c := ui.Prompt(r, w, "  d)elete launchagent  i)mport to TOML  s)kip  q)uit > ")
		switch c {
		case 'd':
			if err := manager.DeleteOrphan(name); err != nil {
				fmt.Fprintf(w, "  error: %v\n", err)
				continue
			}
			fmt.Fprintf(w, "  %s\n\n", ui.Dim.Render("deleted"))
			return false
		case 'i':
			if err := manager.Import(name, service.ImportOpts{Force: false}); err != nil {
				fmt.Fprintf(w, "  error: %v\n", err)
				continue
			}
			fmt.Fprintf(w, "  %s %s\n\n", ui.Dim.Render("imported to"), cfgPath)
			return false
		case 's':
			fmt.Fprintln(w, ui.Dim.Render("  skipped"))
			fmt.Fprintln(w)
			return false
		case 'q':
			return true
		default:
			fmt.Fprintln(w, "  invalid choice")
		}
	}
}

func handleModified(w io.Writer, r *bufio.Reader, idx, total int, name string) bool {
	label := config.Label(name)
	st, _ := manager.Status(name)
	cfgPath, _ := config.ServiceConfigPath(name)
	plistPath, _ := config.PlistPath(name)

	fmt.Fprintf(w, "[%d/%d] %s: %s\n", idx, total, ui.Bold.Render(label), ui.Modified.Render("modified"))
	fmt.Fprintf(w, "  config newer than plist\n")
	fmt.Fprintf(w, "  config:   %s\n", cfgPath)
	fmt.Fprintf(w, "  plist:    %s\n", plistPath)
	if st != nil && st.PID > 0 {
		fmt.Fprintf(w, "  running:  %s\n", ui.Running.Render(fmt.Sprintf("pid %d", st.PID)))
	}
	for {
		c := ui.Prompt(r, w, "  a)pply TOML to plist  i)mport plist over TOML  s)kip  q)uit > ")
		switch c {
		case 'a':
			if err := manager.Apply(name); err != nil {
				fmt.Fprintf(w, "  error: %v\n", err)
				continue
			}
			fmt.Fprintf(w, "  %s\n\n", ui.Dim.Render("applied"))
			return false
		case 'i':
			if !ui.Confirm(bufio.NewReader(os.Stdin), w, "  this overwrites your TOML edits with the plist content. proceed?") {
				fmt.Fprintln(w, "  cancelled")
				continue
			}
			if err := manager.Import(name, service.ImportOpts{Force: true}); err != nil {
				fmt.Fprintf(w, "  error: %v\n", err)
				continue
			}
			fmt.Fprintf(w, "  %s %s\n\n", ui.Dim.Render("imported to"), cfgPath)
			return false
		case 's':
			fmt.Fprintln(w, ui.Dim.Render("  skipped"))
			fmt.Fprintln(w)
			return false
		case 'q':
			return true
		default:
			fmt.Fprintln(w, "  invalid choice")
		}
	}
}
