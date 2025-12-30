package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/will/ws/internal/git"
	"github.com/will/ws/internal/workspace"
)

// PruneCmd handles the 'ws prune' command.
func PruneCmd(args []string) int {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Show what would be removed")
	yes := fs.Bool("yes", false, "Don't prompt for orphan removal")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws prune [--dry-run] [--yes]\n\n")
		fmt.Fprintf(os.Stderr, "Clean up stale worktrees.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
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

	if *dryRun {
		fmt.Println("Dry run - no changes will be made")
		fmt.Println()
	}

	// Prune git worktree metadata
	if !*dryRun {
		if err := git.PruneWorktrees(); err != nil {
			fmt.Fprintf(os.Stderr, "ws: failed to prune worktrees: %v\n", err)
			return 1
		}
		fmt.Println("Pruned git worktree metadata")
	} else {
		fmt.Println("Would prune git worktree metadata")
	}

	// Find orphaned directories in workspace directory
	wsDir := mgr.Config.GetWorkspaceDir(mgr.RepoRoot)
	entries, err := os.ReadDir(wsDir)
	if err != nil {
		// No workspace directory means no orphans
		if os.IsNotExist(err) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "ws: failed to read workspace directory: %v\n", err)
		return 1
	}

	worktrees, err := git.ListWorktrees()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to list worktrees: %v\n", err)
		return 1
	}

	// Build set of valid worktree paths
	validPaths := make(map[string]bool)
	for _, wt := range worktrees {
		validPaths[wt.Path] = true
	}

	// Find orphans
	var orphans []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := fmt.Sprintf("%s/%s", wsDir, entry.Name())
		if !validPaths[path] {
			orphans = append(orphans, path)
		}
	}

	if len(orphans) == 0 {
		fmt.Println()
		fmt.Println("No orphaned directories found.")
		return 0
	}

	fmt.Println()
	fmt.Println("Found orphaned directories:")
	for _, path := range orphans {
		info, _ := os.Stat(path)
		var modTime string
		if info != nil {
			modTime = fmt.Sprintf("last modified %s", info.ModTime().Format("2006-01-02"))
		}
		fmt.Printf("  %s (%s)\n", shortenPath(path), modTime)
	}

	if *dryRun {
		fmt.Println()
		fmt.Println("Would remove these directories (use without --dry-run to remove)")
		return 0
	}

	if !*yes {
		fmt.Println()
		fmt.Print("Remove orphaned directories? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return 0
		}
	}

	// Remove orphans
	for _, path := range orphans {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintf(os.Stderr, "ws: failed to remove %s: %v\n", path, err)
		} else {
			fmt.Printf("Removed: %s\n", shortenPath(path))
		}
	}

	return 0
}
