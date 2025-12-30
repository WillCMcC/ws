package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/will/ws/internal/git"
)

// Config holds the ws configuration.
type Config struct {
	Workspace WorkspaceConfig
	Hooks     HooksConfig
	Status    StatusConfig
}

// WorkspaceConfig holds workspace-related settings.
type WorkspaceConfig struct {
	Directory   string
	DefaultBase string
}

// HooksConfig holds hook-related settings.
type HooksConfig struct {
	PostCreate string
	PreRemove  string
}

// StatusConfig holds status-related settings.
type StatusConfig struct {
	DetectProcesses bool
	AgentProcesses  []string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Workspace: WorkspaceConfig{
			Directory:   "../{repo}-ws",
			DefaultBase: "", // Empty means auto-detect (main or master)
		},
		Hooks: HooksConfig{
			PostCreate: "",
			PreRemove:  "",
		},
		Status: StatusConfig{
			DetectProcesses: true,
			AgentProcesses:  []string{"claude", "opencode", "aider", "codex", "gemini"},
		},
	}
}

// Load loads configuration from files and environment.
func Load() *Config {
	cfg := DefaultConfig()

	// Override with environment variables
	if dir := os.Getenv("WS_DIRECTORY"); dir != "" {
		cfg.Workspace.Directory = dir
	}
	if base := os.Getenv("WS_DEFAULT_BASE"); base != "" {
		cfg.Workspace.DefaultBase = base
	}
	if os.Getenv("WS_NO_HOOKS") == "1" {
		cfg.Hooks.PostCreate = ""
		cfg.Hooks.PreRemove = ""
	}

	return cfg
}

// GetWorkspaceDir returns the resolved workspace directory path.
func (c *Config) GetWorkspaceDir(repoRoot string) string {
	dir := c.Workspace.Directory

	// Replace {repo} placeholder
	repoName := git.RepoName(repoRoot)
	dir = strings.ReplaceAll(dir, "{repo}", repoName)

	// Handle relative paths
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoRoot, dir)
	}

	return filepath.Clean(dir)
}

// GetDefaultBase returns the default base branch.
func (c *Config) GetDefaultBase() string {
	if c.Workspace.DefaultBase != "" {
		return c.Workspace.DefaultBase
	}
	return git.GetDefaultBranch()
}
