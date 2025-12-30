package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/WillCMcC/ws/internal/workspace"
)

// DoneCmd handles the 'ws done' command.
func DoneCmd(args []string) int {
	fs := flag.NewFlagSet("done", flag.ExitOnError)
	force := fs.Bool("force", false, "Remove even with uncommitted changes")
	keepBranch := fs.Bool("keep-branch", false, "Don't delete the branch")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws done <name> [--force] [--keep-branch]\n\n")
		fmt.Fprintf(os.Stderr, "Remove a workspace (worktree + optionally branch).\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  name    Workspace to remove\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "ws: missing workspace name\n")
		fmt.Fprintf(os.Stderr, "    Usage: ws done <name>\n")
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

	if err := mgr.Remove(name, *force, *keepBranch); err != nil {
		// Don't print error again if it's about uncommitted changes
		// (already printed by Remove)
		if err.Error() != "workspace has uncommitted changes" {
			fmt.Fprintf(os.Stderr, "ws: %v\n", err)
		}
		return 1
	}

	return 0
}
