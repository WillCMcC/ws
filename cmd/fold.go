package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/WillCMcC/ws/internal/git"
	"github.com/WillCMcC/ws/internal/workspace"
)

// FoldCmd handles the 'ws fold' command.
func FoldCmd(args []string) int {
	fs := flag.NewFlagSet("fold", flag.ExitOnError)
	noDone := fs.Bool("no-done", false, "Don't remove workspace after folding")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws fold [name] [--no-done]\n\n")
		fmt.Fprintf(os.Stderr, "Rebase workspace onto default branch and merge it in.\n\n")
		fmt.Fprintf(os.Stderr, "If no name is given, uses the current workspace.\n\n")
		fmt.Fprintf(os.Stderr, "Steps performed:\n")
		fmt.Fprintf(os.Stderr, "  1. Rebase workspace branch onto default branch\n")
		fmt.Fprintf(os.Stderr, "  2. Fast-forward merge into default branch\n")
		fmt.Fprintf(os.Stderr, "  3. Remove workspace (unless --no-done)\n\n")
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

	// Determine which workspace to fold
	var wsName string
	var wsPath string

	if fs.NArg() >= 1 {
		// Workspace name provided
		wsName = fs.Arg(0)
		ws, err := mgr.Get(wsName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ws: %v\n", err)
			return 1
		}
		wsPath = ws.Path
	} else {
		// Try to detect current workspace from cwd
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ws: failed to get current directory: %v\n", err)
			return 1
		}

		workspaces, err := mgr.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ws: failed to list workspaces: %v\n", err)
			return 1
		}

		for _, ws := range workspaces {
			if isInOrEqualDir(cwd, ws.Path) {
				wsName = ws.Name
				wsPath = ws.Path
				break
			}
		}

		if wsName == "" {
			fmt.Fprintf(os.Stderr, "ws: not in a workspace\n")
			fmt.Fprintf(os.Stderr, "    Run from within a workspace, or specify: ws fold <name>\n")
			return 1
		}
	}

	defaultBranch := mgr.Config.GetDefaultBase()

	// Check for uncommitted changes
	hasChanges, _, err := git.HasUncommittedChanges(wsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to check workspace status: %v\n", err)
		return 1
	}
	if hasChanges {
		fmt.Fprintf(os.Stderr, "ws: workspace '%s' has uncommitted changes\n", wsName)
		fmt.Fprintf(os.Stderr, "    Commit or stash changes before folding.\n")
		return 1
	}

	fmt.Printf("Folding workspace '%s' into '%s'...\n\n", wsName, defaultBranch)

	// Step 1: Fetch latest default branch in main repo
	fmt.Printf("Fetching latest '%s'...\n", defaultBranch)
	if err := runGitCmd(mgr.RepoRoot, "fetch", "origin", defaultBranch); err != nil {
		// Fetch might fail if no remote, that's ok
		fmt.Printf("  (no remote or fetch failed, continuing with local)\n")
	}

	// Step 2: Rebase workspace onto default branch
	fmt.Printf("Rebasing '%s' onto '%s'...\n", wsName, defaultBranch)
	if err := runGitCmd(wsPath, "rebase", defaultBranch); err != nil {
		fmt.Fprintf(os.Stderr, "\nws: rebase failed\n")
		fmt.Fprintf(os.Stderr, "    Resolve conflicts in %s, then run:\n", wsPath)
		fmt.Fprintf(os.Stderr, "      cd %s && git rebase --continue\n", wsPath)
		fmt.Fprintf(os.Stderr, "    Or abort with: git rebase --abort\n")
		return 1
	}

	// Step 3: Go to main repo and merge
	fmt.Printf("Merging '%s' into '%s'...\n", wsName, defaultBranch)

	// Checkout default branch in main repo
	if err := runGitCmd(mgr.RepoRoot, "checkout", defaultBranch); err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to checkout '%s': %v\n", defaultBranch, err)
		return 1
	}

	// Fast-forward merge the workspace branch
	if err := runGitCmd(mgr.RepoRoot, "merge", "--ff-only", wsName); err != nil {
		fmt.Fprintf(os.Stderr, "ws: merge failed (not fast-forward)\n")
		fmt.Fprintf(os.Stderr, "    The rebase may not have completed properly.\n")
		return 1
	}

	fmt.Printf("\nSuccessfully merged '%s' into '%s'\n", wsName, defaultBranch)

	// Step 4: Clean up workspace (unless --no-done)
	if !*noDone {
		fmt.Printf("Cleaning up workspace...\n")
		// Use force since we just merged everything
		if err := mgr.Remove(wsName, true, false); err != nil {
			fmt.Fprintf(os.Stderr, "ws: warning: failed to remove workspace: %v\n", err)
		}
	}

	fmt.Printf("\nDone! Don't forget to push:\n")
	fmt.Printf("  git push origin %s\n", defaultBranch)

	return 0
}

// runGitCmd runs a git command in the specified directory.
func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isInOrEqualDir checks if path is inside or equal to dir.
func isInOrEqualDir(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	if absPath == absDir {
		return true
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
