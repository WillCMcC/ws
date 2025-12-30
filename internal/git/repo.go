package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindRepoRoot returns the root directory of the current git repository.
func FindRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(output)), nil
}

// RepoName returns the basename of the repository directory.
func RepoName(repoRoot string) string {
	return filepath.Base(repoRoot)
}

// GetDefaultBranch returns the default branch name (main or master).
func GetDefaultBranch() string {
	// Try to get the default branch from git config
	cmd := exec.Command("git", "config", "--get", "init.defaultBranch")
	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		if branch != "" {
			return branch
		}
	}

	// Check if main exists
	cmd = exec.Command("git", "rev-parse", "--verify", "main")
	if err := cmd.Run(); err == nil {
		return "main"
	}

	// Check if master exists
	cmd = exec.Command("git", "rev-parse", "--verify", "master")
	if err := cmd.Run(); err == nil {
		return "master"
	}

	// Default to main
	return "main"
}

// BranchExists checks if a branch exists.
func BranchExists(name string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", name)
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current branch name.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
