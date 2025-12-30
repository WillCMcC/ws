package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/WillCMcC/ws/internal/workspace"
)

// GoCmd handles the 'ws go' command.
func GoCmd(args []string) int {
	fs := flag.NewFlagSet("go", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws go <name>\n\n")
		fmt.Fprintf(os.Stderr, "Navigate to a workspace directory.\n\n")
		fmt.Fprintf(os.Stderr, "Note: This command prints the path. Use shell integration\n")
		fmt.Fprintf(os.Stderr, "for the 'cd' to work. Run 'ws init' to set up.\n")
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "ws: missing workspace name\n")
		fmt.Fprintf(os.Stderr, "    Usage: ws go <name>\n")
		return 1
	}

	name := fs.Arg(0)

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: %v\n", err)
		if err.Error() == "not a git repository" {
			fmt.Fprintf(os.Stderr, "    Run this command from within a git repository.\n")
			return 2
		}
		return 1
	}

	path, err := mgr.GetPath(name)
	if err != nil {
		// Show helpful error with available workspaces
		fmt.Fprintf(os.Stderr, "ws: workspace '%s' not found\n", name)
		workspaces, listErr := mgr.List()
		if listErr == nil && len(workspaces) > 0 {
			fmt.Fprintf(os.Stderr, "    Available workspaces:\n")
			for _, ws := range workspaces {
				fmt.Fprintf(os.Stderr, "      %s\n", ws.Name)
			}
		}
		fmt.Fprintf(os.Stderr, "    To create it: ws new %s\n", name)
		return 3
	}

	// Print just the path to stdout (shell integration will use this)
	fmt.Println(path)
	return 0
}
