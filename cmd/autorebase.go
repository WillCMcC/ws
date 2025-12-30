package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AutoRebaseCmd handles the 'ws auto-rebase' command.
// It checks for rebase state and signals shell to start agent with rebase prompt.
func AutoRebaseCmd(args []string) int {
	fs := flag.NewFlagSet("auto-rebase", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws auto-rebase\n\n")
		fmt.Fprintf(os.Stderr, "Start your agent to help resolve rebase conflicts.\n\n")
		fmt.Fprintf(os.Stderr, "Run this when 'ws fold' fails due to merge conflicts.\n")
		fmt.Fprintf(os.Stderr, "The agent will help resolve conflicts, then run 'ws fold' again.\n\n")
		fmt.Fprintf(os.Stderr, "Example workflow:\n")
		fmt.Fprintf(os.Stderr, "  ws fold              # fails with conflicts\n")
		fmt.Fprintf(os.Stderr, "  ws auto-rebase       # agent helps resolve\n")
		fmt.Fprintf(os.Stderr, "  ws fold              # try again after conflicts resolved\n")
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Check if we're in a rebase state
	if !isRebaseInProgress() {
		fmt.Fprintf(os.Stderr, "ws: no rebase in progress\n")
		fmt.Fprintf(os.Stderr, "    Run 'ws fold' first. If it fails with conflicts, run 'ws auto-rebase'.\n")
		return 1
	}

	// Show current conflict status
	fmt.Println("Rebase in progress. Conflicted files:")
	showConflictedFiles()
	fmt.Println()
	fmt.Println("Starting agent to help resolve conflicts...")
	fmt.Println()

	// Exit 0 to signal shell function to start agent
	return 0
}

// isRebaseInProgress checks if there's an active rebase
func isRebaseInProgress() bool {
	// Check for .git/rebase-merge or .git/rebase-apply directories
	gitDir := findGitDir()
	if gitDir == "" {
		return false
	}

	rebaseMerge := gitDir + "/rebase-merge"
	rebaseApply := gitDir + "/rebase-apply"

	if _, err := os.Stat(rebaseMerge); err == nil {
		return true
	}
	if _, err := os.Stat(rebaseApply); err == nil {
		return true
	}
	return false
}

// findGitDir returns the .git directory path
func findGitDir() string {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// showConflictedFiles prints files with conflicts
func showConflictedFiles() {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
