package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/will/ws/internal/workspace"
)

// HomeCmd handles the 'ws home' command.
func HomeCmd(args []string) int {
	fs := flag.NewFlagSet("home", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws home\n\n")
		fmt.Fprintf(os.Stderr, "Navigate to the main repository directory.\n")
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: %v\n", err)
		if err.Error() == "not a git repository" {
			fmt.Fprintf(os.Stderr, "    Run this command from within a git repository.\n")
			return 2
		}
		return 1
	}

	// Print just the path to stdout (shell integration will use this)
	fmt.Println(mgr.RepoRoot)
	return 0
}
