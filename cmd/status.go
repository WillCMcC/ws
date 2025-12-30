package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/will/ws/internal/git"
	"github.com/will/ws/internal/workspace"
)

// StatusCmd handles the 'ws status' command.
func StatusCmd(args []string) int {
	fs := flag.NewFlagSet("status", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws status\n\n")
		fmt.Fprintf(os.Stderr, "Show detailed status of all workspaces.\n")
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

	workspaces, err := mgr.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to list workspaces: %v\n", err)
		return 1
	}

	if len(workspaces) == 0 {
		fmt.Println("No workspaces found.")
		fmt.Println()
		fmt.Println("Create one with: ws new <name>")
		return 0
	}

	defaultBase := mgr.Config.GetDefaultBase()

	for i, ws := range workspaces {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("%s\n", ws.Name)
		fmt.Printf("  Path:     %s\n", shortenPath(ws.Path))
		fmt.Printf("  Branch:   %s", ws.Branch)

		// Show commits ahead
		if ahead, err := git.GetCommitsAhead(ws.Path, defaultBase); err == nil && ahead > 0 {
			fmt.Printf(" (%d commits ahead of %s)", ahead, defaultBase)
		}
		fmt.Println()

		// Show status
		hasChanges, status, err := git.HasUncommittedChanges(ws.Path)
		if err == nil {
			if hasChanges {
				lines := countLines(status)
				fmt.Printf("  Status:   %d file(s) modified\n", lines)
			} else {
				fmt.Printf("  Status:   clean\n")
			}
		}

		// Show last commit
		if msg, when, err := git.GetLastCommit(ws.Path); err == nil {
			// Truncate message if too long
			if len(msg) > 40 {
				msg = msg[:37] + "..."
			}
			fmt.Printf("  Last:     \"%s\" (%s)\n", msg, when)
		}

		// Process detection placeholder
		fmt.Printf("  Process:  none detected\n")
	}

	return 0
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	// Count last line if no trailing newline
	if len(s) > 0 && s[len(s)-1] != '\n' {
		count++
	}
	return count
}
