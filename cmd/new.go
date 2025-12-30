package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/will/ws/internal/workspace"
)

// NewCmd handles the 'ws new' command.
func NewCmd(args []string) int {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	from := fs.String("from", "", "Base ref to branch from")
	noHooks := fs.Bool("no-hooks", false, "Skip post-create hooks")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws new <name> [--from <ref>] [--no-hooks]\n\n")
		fmt.Fprintf(os.Stderr, "Create a new workspace (worktree + branch).\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  name    Workspace name (becomes branch and directory name)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "ws: missing workspace name\n")
		fmt.Fprintf(os.Stderr, "    Usage: ws new <name>\n")
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

	if err := mgr.Create(name, *from, *noHooks); err != nil {
		fmt.Fprintf(os.Stderr, "ws: %v\n", err)
		return 1
	}

	return 0
}
