package main

import (
	"fmt"
	"os"

	"github.com/evnoj/launchdude/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "launchdude:", err)
		os.Exit(1)
	}
}
