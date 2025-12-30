package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/WillCMcC/ws/internal/config"
	"github.com/WillCMcC/ws/internal/workspace"
)

// EzCmd handles the 'ws ez' command.
// It creates a workspace and outputs the path + agent command for shell integration.
func EzCmd(args []string) int {
	fs := flag.NewFlagSet("ez", flag.ExitOnError)
	from := fs.String("from", "", "Base ref to branch from")
	noHooks := fs.Bool("no-hooks", false, "Skip post-create hooks")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws ez <name> [--from <ref>] [--no-hooks]\n\n")
		fmt.Fprintf(os.Stderr, "Create a workspace, navigate to it, and start your agent.\n\n")
		fmt.Fprintf(os.Stderr, "Configure the agent command with:\n")
		fmt.Fprintf(os.Stderr, "  ws config set agent_cmd \"claude --dangerously-skip-permissions\"\n\n")
		fmt.Fprintf(os.Stderr, "Default: claude\n\n")
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
		fmt.Fprintf(os.Stderr, "    Usage: ws ez <name>\n")
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

// GetAgentCmd returns the configured agent command.
func GetAgentCmd() string {
	cfg := config.Load()
	return cfg.Agent.Cmd
}
