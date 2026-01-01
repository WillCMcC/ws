package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WillCMcC/ws/internal/config"
	"github.com/WillCMcC/ws/internal/git"
)

// Workspace represents a managed workspace.
type Workspace struct {
	Name     string
	Path     string
	Branch   string
	Modified time.Time
}

// Manager handles workspace operations.
type Manager struct {
	RepoRoot string
	Config   *config.Config
}

// NewManager creates a new workspace manager.
func NewManager() (*Manager, error) {
	repoRoot, err := git.FindRepoRoot()
	if err != nil {
		return nil, err
	}

	return &Manager{
		RepoRoot: repoRoot,
		Config:   config.Load(),
	}, nil
}

// Create creates a new workspace.
func (m *Manager) Create(name, base string, noHooks bool) error {
	if base == "" {
		base = m.Config.GetDefaultBase()
	}

	// Check if branch already exists
	if git.BranchExists(name) {
		return fmt.Errorf("branch '%s' already exists", name)
	}

	// Get workspace directory
	wsDir := m.Config.GetWorkspaceDir(m.RepoRoot)
	wsPath := filepath.Join(wsDir, name)

	// Check if path already exists
	if _, err := os.Stat(wsPath); err == nil {
		return fmt.Errorf("directory '%s' already exists", wsPath)
	}

	// Ensure workspace directory exists
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create worktree
	if err := git.CreateWorktree(wsPath, name, base); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Run post-create hook if configured and not disabled
	if !noHooks && m.Config.Hooks.PostCreate != "" {
		if err := runHook(m.Config.Hooks.PostCreate, wsPath, name); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: post-create hook failed: %v\n", err)
		}
	}

	// Print success message (minimal - shell function handles cd)
	fmt.Printf("Created workspace: %s (from %s)\n", name, base)

	return nil
}

// List returns all managed workspaces.
func (m *Manager) List() ([]Workspace, error) {
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return nil, err
	}

	wsDir := m.Config.GetWorkspaceDir(m.RepoRoot)
	var workspaces []Workspace

	for _, wt := range worktrees {
		// Only include worktrees in our workspace directory
		if !isInDirectory(wt.Path, wsDir) {
			continue
		}

		ws := Workspace{
			Name:   filepath.Base(wt.Path),
			Path:   wt.Path,
			Branch: wt.Branch,
		}

		// Get modification time
		if info, err := os.Stat(wt.Path); err == nil {
			ws.Modified = info.ModTime()
		}

		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// Get returns a workspace by name.
func (m *Manager) Get(name string) (*Workspace, error) {
	workspaces, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, ws := range workspaces {
		if ws.Name == name {
			return &ws, nil
		}
	}

	return nil, fmt.Errorf("workspace '%s' not found", name)
}

// Remove removes a workspace.
func (m *Manager) Remove(name string, force, keepBranch bool) error {
	ws, err := m.Get(name)
	if err != nil {
		return err
	}

	// Check for uncommitted changes
	hasChanges, status, err := git.HasUncommittedChanges(ws.Path)
	if err != nil {
		return fmt.Errorf("failed to check workspace status: %w", err)
	}

	if hasChanges && !force {
		fmt.Printf("Workspace %s has uncommitted changes:\n", name)
		fmt.Print(status)
		fmt.Println()
		fmt.Printf("ws done %s --force  # remove anyway\n", name)
		fmt.Printf("git stash           # or stash changes first\n")
		return fmt.Errorf("workspace has uncommitted changes")
	}

	// Run pre-remove hook if configured
	if m.Config.Hooks.PreRemove != "" {
		if err := runHook(m.Config.Hooks.PreRemove, ws.Path, name); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: pre-remove hook failed: %v\n", err)
		}
	}

	// Remove worktree
	if err := git.RemoveWorktree(ws.Path, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Delete branch unless --keep-branch
	if !keepBranch {
		if err := git.DeleteBranch(name, force); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete branch: %v\n", err)
		}
	}

	fmt.Printf("Removed workspace: %s\n", name)
	if !keepBranch {
		fmt.Printf("  Branch %s deleted\n", name)
	}

	return nil
}

// GetPath returns the path to a workspace.
func (m *Manager) GetPath(name string) (string, error) {
	ws, err := m.Get(name)
	if err != nil {
		return "", err
	}
	return ws.Path, nil
}

// GetMainWorktree returns info about the main worktree.
func (m *Manager) GetMainWorktree() (string, string, error) {
	branch, err := git.GetCurrentBranch()
	if err != nil {
		branch = "unknown"
	}
	return m.RepoRoot, branch, nil
}

// isInDirectory checks if path is inside dir.
func isInDirectory(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && rel != ".." && !startsWithDotDot(rel)
}

func startsWithDotDot(path string) bool {
	return len(path) >= 2 && path[0:2] == ".."
}
