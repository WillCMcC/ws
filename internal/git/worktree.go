package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree.
type Worktree struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
}

// CreateWorktree creates a new worktree with a new branch.
func CreateWorktree(path, branch, base string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branch, path, base)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveWorktree removes a worktree.
func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DeleteBranch deletes a branch.
func DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ListWorktrees returns all worktrees for the repository.
func ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseWorktreePorcelain(string(output))
}

// parseWorktreePorcelain parses the porcelain output of git worktree list.
func parseWorktreePorcelain(output string) ([]Worktree, error) {
	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			// Branch is in format refs/heads/branchname
			branch := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(branch, "refs/heads/")
		} else if line == "bare" && current != nil {
			current.Bare = true
		}
	}

	// Don't forget the last worktree if there's no trailing newline
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, scanner.Err()
}

// PruneWorktrees runs git worktree prune.
func PruneWorktrees() error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetWorktreeStatus returns the git status for a worktree.
func GetWorktreeStatus(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// HasUncommittedChanges checks if a worktree has uncommitted changes.
func HasUncommittedChanges(path string) (bool, string, error) {
	status, err := GetWorktreeStatus(path)
	if err != nil {
		return false, "", err
	}
	return status != "", status, nil
}

// GetCommitsAhead returns how many commits the branch is ahead of base.
func GetCommitsAhead(path, base string) (int, error) {
	cmd := exec.Command("git", "-C", path, "rev-list", "--count", fmt.Sprintf("%s..HEAD", base))
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var count int
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count, err
}

// GetLastCommit returns the last commit info for a worktree.
func GetLastCommit(path string) (string, string, error) {
	cmd := exec.Command("git", "-C", path, "log", "-1", "--format=%s|%ar")
	output, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected log format")
	}
	return parts[0], parts[1], nil
}

// WorktreeExists checks if a worktree exists at the given path.
func WorktreeExists(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	worktrees, err := ListWorktrees()
	if err != nil {
		return false
	}
	for _, wt := range worktrees {
		if wt.Path == absPath {
			return true
		}
	}
	return false
}
