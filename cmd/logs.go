package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/evnoj/launchdude/internal/config"
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsLines  int
	logsStdout bool
	logsStderr bool
)

var logsCmd = &cobra.Command{
	Use:               "logs NAME",
	Short:             "Tail the stdout and stderr logs of a service",
	Long:              "Shells out to `tail`. By default shows both streams with file headers. Use --stdout or --stderr to pick one. Use -f to follow.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: serviceNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := config.ValidateName(name); err != nil {
			return err
		}
		svc, err := config.LoadService(name)
		if err != nil {
			return err
		}

		var paths []string
		switch {
		case logsStdout && logsStderr, !logsStdout && !logsStderr:
			out, err := svc.ResolveStdoutPath(name)
			if err != nil {
				return err
			}
			errp, err := svc.ResolveStderrPath(name)
			if err != nil {
				return err
			}
			paths = []string{out, errp}
		case logsStdout:
			out, err := svc.ResolveStdoutPath(name)
			if err != nil {
				return err
			}
			paths = []string{out}
		case logsStderr:
			errp, err := svc.ResolveStderrPath(name)
			if err != nil {
				return err
			}
			paths = []string{errp}
		}

		for _, p := range paths {
			if _, err := os.Stat(p); os.IsNotExist(err) {
				fmt.Fprintf(cmd.OutOrStderr(), "warning: %s does not exist yet (service may not have run)\n", p)
			}
		}

		tailArgs := []string{"-n", strconv.Itoa(logsLines)}
		if logsFollow {
			tailArgs = append(tailArgs, "-f")
		}
		tailArgs = append(tailArgs, paths...)
		c := exec.Command("tail", tailArgs...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow appended output")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "number of lines to show from each file")
	logsCmd.Flags().BoolVar(&logsStdout, "stdout", false, "show only the stdout stream")
	logsCmd.Flags().BoolVar(&logsStderr, "stderr", false, "show only the stderr stream")
}
