# ws - Workspace Manager for Parallel Agent Development

`ws` is a minimal CLI tool that manages git worktrees for running multiple coding agents in parallel on the same codebase. It's agent-agnostic, works with any CLI tool (Claude Code, Aider, Codex, Gemini CLI, etc.), and leaves all git operations (commit, merge, push) to you.

## Why?

When working with AI coding agents, you often want to run multiple tasks in parallel:
- One agent fixing a bug
- Another adding a feature
- A third refactoring tests

Git worktrees let each agent work in its own isolated directory with its own branch. `ws` makes managing these worktrees effortless.

## Installation

### From Source

```bash
go install github.com/will/ws@latest
```

### Manual Build

```bash
git clone https://github.com/will/ws.git
cd ws
make
sudo make install
```

### Shell Integration (Required for `ws go`)

Run `ws init` to set up shell integration, or manually add to your shell config:

<details>
<summary>Bash (~/.bashrc)</summary>

```bash
ws() {
    if [[ "$1" == "go" && -n "$2" ]]; then
        local target
        target=$(command ws go "$2" 2>/dev/null)
        local exit_code=$?
        if [[ $exit_code -eq 0 && -n "$target" && -d "$target" ]]; then
            cd "$target" || return 1
        else
            command ws go "$2"
            return $exit_code
        fi
    else
        command ws "$@"
    fi
}
```
</details>

<details>
<summary>Zsh (~/.zshrc)</summary>

```zsh
ws() {
    if [[ "$1" == "go" && -n "$2" ]]; then
        local target
        target=$(command ws go "$2" 2>/dev/null)
        local exit_code=$?
        if [[ $exit_code -eq 0 && -n "$target" && -d "$target" ]]; then
            cd "$target"
        else
            command ws go "$2"
            return $exit_code
        fi
    else
        command ws "$@"
    fi
}
```
</details>

<details>
<summary>Fish (~/.config/fish/conf.d/ws.fish)</summary>

```fish
function ws
    if test "$argv[1]" = "go" -a -n "$argv[2]"
        set -l target (command ws go $argv[2] 2>/dev/null)
        set -l exit_code $status
        if test $exit_code -eq 0 -a -n "$target" -a -d "$target"
            cd $target
        else
            command ws go $argv[2]
            return $exit_code
        end
    else
        command ws $argv
    end
end
```
</details>

## Quick Start

```bash
cd ~/projects/myapp

# Create a workspace (automatically cd's into it)
ws new auth-feature
claude  # or aider, codex, etc.

# In another terminal, create another workspace
cd ~/projects/myapp
ws new fix-bug
claude

# Check status of all workspaces
ws list
ws status

# Navigate between workspaces
ws go auth-feature    # go to workspace
ws home               # go back to main repo

# When done with a workspace
ws home
git merge auth-feature  # merge the work
ws done auth-feature    # clean up
```

## Commands

### `ws new <name> [--from <ref>]`

Create a new workspace (worktree + branch).

```bash
ws new auth-feature              # Branch from default (main/master)
ws new bugfix --from develop     # Branch from specific ref
ws new experiment --no-hooks     # Skip post-create hooks
```

### `ws list [--json] [--quiet]`

List all workspaces.

```bash
ws list           # Table format
ws list --json    # JSON output
ws list --quiet   # Just names (for scripting)
```

### `ws go <name>`

Navigate to a workspace directory (requires shell integration).

```bash
ws go auth-feature
```

### `ws done <name> [--force] [--keep-branch]`

Remove a workspace.

```bash
ws done auth-feature              # Remove worktree and branch
ws done auth-feature --keep-branch # Keep the branch
ws done auth-feature --force      # Remove even with uncommitted changes
```

### `ws status`

Show detailed status of all workspaces.

```bash
ws status
```

Output:
```
auth-feature
  Path:     ~/myapp-ws/auth-feature
  Branch:   auth-feature (3 commits ahead of main)
  Status:   2 file(s) modified
  Last:     "Add login form" (2 hours ago)
  Process:  none detected
```

### `ws prune [--dry-run] [--yes]`

Clean up stale worktrees and orphaned directories.

```bash
ws prune            # Interactive
ws prune --dry-run  # Show what would be removed
ws prune --yes      # Don't prompt
```

### `ws init [--shell <type>]`

Set up shell integration.

```bash
ws init             # Auto-detect shell
ws init --shell zsh # Specify shell
```

## Directory Layout

By default, workspaces are created in a sibling directory:

```
~/projects/
├── myapp/                    # Main repository
│   ├── .git/
│   └── src/
└── myapp-ws/                 # Workspace directory
    ├── auth-feature/         # Worktree for auth-feature branch
    └── fix-bug/              # Worktree for fix-bug branch
```

## Configuration

### Environment Variables

```bash
WS_DIRECTORY="../workspaces"  # Override workspace directory
WS_DEFAULT_BASE="develop"     # Override default base branch
WS_NO_HOOKS="1"               # Disable all hooks
```

## Design Principles

1. **Worktrees are workspaces** — each gets its own directory, branch, and isolated filesystem
2. **Agents are just processes** — ws doesn't know or care what runs in a workspace
3. **Git stays yours** — ws creates branches/worktrees; you commit, push, merge
4. **No daemon** — stateless CLI that reads from git/filesystem each invocation
5. **Shell integration for navigation** — `ws go <name>` changes directory via shell function

## License

MIT
