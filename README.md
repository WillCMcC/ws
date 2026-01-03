# ws - Workspace Manager for Parallel Agent Development

`ws` helps launch multiple agents into their own workspace to run feature dev in parallel.

## Quick start

Bash one liner:

```bash
git clone https://github.com/WillCMcC/ws.git && cd ws && make && sudo make install && ws init
```

`ws new` creates a fresh workspace (branch and git worktree) to isolate development.

`ws ez` does the above but runs `claude` automatically (you lazy bastard you)

Once feature work is done, manage git yourself:

`git add .`

`git commit -m 'my message'`

then

`ws fold` rebases back to main (or master, configurable via `ws config`)

If you hit merge conflicts, `ws auto-rebase` launches your agent to help resolve them.

## The docs

`ws` is a minimal CLI tool that manages git worktrees for running multiple coding agents in parallel on the same codebase. It's agent-agnostic, works with any CLI tool (Claude Code, Aider, Codex, Gemini CLI, etc.), and leaves all git operations (commit, merge, push) to you.

## Why?

When working with AI coding agents, you often want to run multiple tasks in parallel:

- One agent fixing a bug
- Another adding a feature
- A third refactoring tests

Git worktrees let each agent work in its own isolated directory with its own branch. `ws` makes managing these worktrees effortless.

## Installation

### Quick Install (macOS/Linux)

```bash
git clone https://github.com/WillCMcC/ws.git && cd ws && make && sudo make install && ws init
```

### From Source

```bash
go install github.com/WillCMcC/ws@latest
```

### Manual Build

```bash
git clone https://github.com/WillCMcC/ws.git
cd ws
make
sudo make install
```

## Configuration

### Using `ws config` (Recommended)

Launches an interactive config. Power users can config via the cli.

```bash
ws config set agent_cmd "claude --dangerously-skip-permissions"
ws config set default_base "develop"
ws config list
```

Config is stored in `~/.config/ws/config`.

### Shell Integration (Required for navigation)

Run `ws init` to automatically set up shell integration. It will:

- Add the shell function to your config
- Copy the `source` command to your clipboard

Or manually add to your shell config:

<details>
<summary>Bash (~/.bashrc)</summary>

```bash
source /path/to/ws-cli/scripts/shell/ws.bash
```

Or see [scripts/shell/ws.bash](scripts/shell/ws.bash) for the full function.

</details>

<details>
<summary>Zsh (~/.zshrc)</summary>

```zsh
source /path/to/ws-cli/scripts/shell/ws.zsh
```

Or see [scripts/shell/ws.zsh](scripts/shell/ws.zsh) for the full function.

</details>

<details>
<summary>Fish (~/.config/fish/conf.d/ws.fish)</summary>

```fish
source /path/to/ws-cli/scripts/shell/ws.fish
```

Or see [scripts/shell/ws.fish](scripts/shell/ws.fish) for the full function.

</details>

## Quick Start

```bash
cd ~/projects/myapp

# The easy way: create workspace and start your agent in one command
ws ez auth-feature    # creates workspace, cd's into it, runs your agent

# Or step by step:
ws new auth-feature   # creates workspace and cd's into it
claude                # start your agent manually

# Check status of all workspaces (detects running agents!)
ws list
ws status

# Navigate between workspaces
ws go auth-feature    # go to workspace
ws home               # go back to main repo

# When done with a workspace''
git add .
git commit -m 'commit msg'
ws home
git merge auth-feature  # merge the work
ws done auth-feature    # clean up

OR

ws fold                 # rebase off of master and merge
```

## Commands

| Command          | Aliases        | Description                      |
| ---------------- | -------------- | -------------------------------- |
| `ws new <name>`  |                | Create a new workspace           |
| `ws ez <name>`   |                | Create workspace and start agent |
| `ws list`        | `ls`           | List all workspaces              |
| `ws go <name>`   |                | Navigate to a workspace          |
| `ws home`        |                | Navigate to main repository      |
| `ws done <name>` | `rm`, `remove` | Remove a workspace               |
| `ws fold [name]` |                | Rebase and merge workspace       |
| `ws auto-rebase` |                | Agent helps resolve rebase conflicts |
| `ws status`      | `st`           | Show detailed workspace status   |
| `ws prune`       |                | Clean up stale worktrees         |
| `ws init`        |                | Set up shell integration         |
| `ws config`      |                | Manage configuration             |

### `ws new <name> [--from <ref>]`

Create a new workspace and navigate to it.

```bash
ws new auth-feature              # Branch from default (main/master)
ws new bugfix --from develop     # Branch from specific ref
ws new experiment --no-hooks     # Skip post-create hooks
```

### `ws ez <name> [--from <ref>]`

Create workspace, navigate to it, and start your agent. The ultimate one-liner.

```bash
# First, configure your agent command (one-time setup):
ws config set agent_cmd "claude --dangerously-skip-permissions"

# Then use ez to create + cd + run agent:
ws ez auth-feature
```

### `ws list [--json] [--quiet]`

List all workspaces.

```bash
ws list           # Table format
ws list --json    # JSON output
ws list --quiet   # Just names (for scripting)
```

### `ws go <name>`

Navigate to a workspace directory.

```bash
ws go auth-feature
```

### `ws home`

Navigate back to the main repository directory.

```bash
ws home
```

### `ws done <name> [--force] [--keep-branch]`

Remove a workspace.

```bash
ws done auth-feature              # Remove worktree and branch
ws done auth-feature --keep-branch # Keep the branch
ws done auth-feature --force      # Remove even with uncommitted changes
```

### `ws fold [name] [--no-done]`

Rebase workspace onto the default branch and merge it in. The complete workflow for finishing a feature.

```bash
ws fold                  # Fold current workspace
ws fold auth-feature     # Fold specific workspace
ws fold --no-done        # Keep workspace after folding
```

This command:

1. Rebases your workspace branch onto the latest default branch
2. Fast-forward merges into the default branch
3. Cleans up the workspace (unless `--no-done`)
4. Returns you to the main repo

### `ws auto-rebase`

When `ws fold` fails due to merge conflicts, run this to get agent help:

```bash
ws fold              # fails with conflicts
ws auto-rebase       # agent resolves conflicts
ws fold              # try again
```

The agent receives a prompt to examine conflicted files and resolve them, then run `git rebase --continue`.

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
  Process:  claude (pid 12345)
```

The process detection automatically finds running agents (claude, aider, codex, etc.) in each workspace.

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

### `ws config <subcommand>`

Manage ws configuration.

```bash
ws config set agent_cmd "claude --dangerously-skip-permissions"
ws config get agent_cmd
ws config list              # Show all settings
ws config path              # Show config file location
```

Available keys:

- `agent_cmd` - Command to run with `ws ez`
- `default_base` - Default base branch for new workspaces
- `directory` - Workspace directory pattern

## Directory Layout

By default, workspaces are created in `.worktrees/{repo}` within your repository:

```
~/projects/myapp/             # Main repository
├── .git/
├── src/
└── .worktrees/               # Workspace directory (gitignored)
    └── myapp/                # Repository name
        ├── auth-feature/     # Worktree for auth-feature branch
        └── fix-bug/          # Worktree for fix-bug branch
```

This keeps worktrees organized and prevents cluttering the parent directory. The `.worktrees` directory is automatically added to `.gitignore` if needed.

### Environment Variables

Environment variables override config file settings:

```bash
WS_AGENT_CMD="claude --dangerously-skip-permissions"  # Agent for 'ws ez'
WS_DIRECTORY=".worktrees/{repo}"  # Override workspace directory (default)
WS_DEFAULT_BASE="develop"         # Override default base branch
WS_NO_HOOKS="1"                   # Disable all hooks
```

## Design Principles

1. **Worktrees are workspaces** — each gets its own directory, branch, and isolated filesystem
2. **Agents are just processes** — ws doesn't know or care what runs in a workspace
3. **Git stays yours** — ws creates branches/worktrees; you commit, push, merge
4. **No daemon** — stateless CLI that reads from git/filesystem each invocation
5. **Shell integration for navigation** — `ws go <name>` changes directory via shell function

## License

Fork me baby idgaf
