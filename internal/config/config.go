package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/WillCMcC/ws/internal/git"
)

// Config holds the ws configuration.
type Config struct {
	Workspace WorkspaceConfig
	Hooks     HooksConfig
	Status    StatusConfig
	Agent     AgentConfig
}

// WorkspaceConfig holds workspace-related settings.
type WorkspaceConfig struct {
	Directory   string
	DefaultBase string
}

// AgentConfig holds agent-related settings.
type AgentConfig struct {
	Cmd string // Command to run for 'ws ez'
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
			Directory:   ".worktrees/{repo}",
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
		Agent: AgentConfig{
			Cmd: "claude", // Default agent command
		},
	}
}

// Load loads configuration from files and environment.
func Load() *Config {
	cfg := DefaultConfig()

	// Load from config file first
	fileConfig := loadConfigFile()

	// Apply config file values
	if dir, ok := fileConfig["directory"]; ok && dir != "" {
		cfg.Workspace.Directory = dir
	}
	if base, ok := fileConfig["default_base"]; ok && base != "" {
		cfg.Workspace.DefaultBase = base
	}
	if agentCmd, ok := fileConfig["agent_cmd"]; ok && agentCmd != "" {
		cfg.Agent.Cmd = agentCmd
	}

	// Override with environment variables (highest priority)
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
	if agentCmd := os.Getenv("WS_AGENT_CMD"); agentCmd != "" {
		cfg.Agent.Cmd = agentCmd
	}

	return cfg
}

// loadConfigFile reads the config file from ~/.config/ws/config
func loadConfigFile() map[string]string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(home, ".config", "ws", "config")
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	config := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return config
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
