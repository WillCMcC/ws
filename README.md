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
| `ws queue`       |                | Interactive task queue manager (TUI) |
| `ws queue-gui`   |                | Graphical task queue manager with menu bar |

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

### `ws queue`

Interactive TUI for managing a task queue. This GUI tool automates the complete workflow:

1. Add tasks to a queue
2. Run `ws ez <task>` for each task automatically
3. Validate changes with git diff preview
4. Commit and `ws fold` on confirmation
5. Handle merge conflicts with interactive `ws auto-rebase`

**Features:**

- **Queue Management**: Add, view, and organize development tasks
- **Automated Workflow**: Automatically creates workspaces and starts agents for queued tasks
- **Validation View**: Preview git changes before committing
- **Conflict Resolution**: Interactive interface for handling merge conflicts during fold
- **Status Tracking**: Visual indicators for task state (queued, running, validating, conflict, completed, failed)

**Usage:**

```bash
# Launch the interactive queue manager
ws queue

# In the queue UI:
# a - add new task
# n - process next pending task
# ↑/↓ or j/k - navigate tasks
# enter - view task details
# c - clear completed tasks
# d - delete selected task
# q - quit

# Task workflow:
# 1. Add task: Press 'a', enter task name and description
# 2. Start task: Press 'n' to process next task (runs ws ez <taskname>)
# 3. Work on task: Agent helps you implement the feature
# 4. Validate: Press 'v' to preview git changes
# 5. Commit: Press 'y' to accept, enter commit message
# 6. Fold: Automatically runs ws fold to merge back to main
# 7. Handle conflicts: If conflicts occur, press 'a' for auto-rebase assistance
```

**Task States:**

- **queued** - Task waiting to be processed
- **running** - Workspace created, agent working on task
- **validating** - Ready for review and commit
- **conflict** - Merge conflicts detected during fold
- **completed** - Successfully folded and merged
- **failed** - Error occurred during processing

**Example Workflow:**

```bash
# Start the queue manager
ws queue

# Add multiple tasks
Press 'a': "auth-feature" -> "Add user authentication"
Press 'a': "fix-api-bug" -> "Fix timeout in API endpoint"
Press 'a': "update-docs" -> "Update README with new features"

# Process first task
Press 'n' - Creates workspace and starts agent for "auth-feature"
# Work with agent on the task...

# When ready, validate changes
Press 'enter' to view task details
Press 'v' to validate and see git diff

# Commit and fold
Press 'y' to accept changes
Enter commit message: "Add JWT authentication system"
# Automatically runs: git add, git commit, ws fold

# Task complete! Process next task
Press 'n' - Starts "fix-api-bug"
```

**Queue Persistence:**

The queue is automatically saved to `~/.config/ws/queue.json` and persists across sessions. You can close and reopen `ws queue` at any time.

### `ws queue-gui`

**Graphical User Interface** for task queue management with macOS menu bar integration.

All the power of `ws queue` with a modern GUI:

**Features:**
- 🖥️ **Cross-platform GUI** - Built with Fyne framework
- 📍 **macOS Menu Bar** - System tray integration for quick access
- 🌟 **Dream Feature** - AI-powered feature suggestions using deep codebase analysis
- 🎨 **Visual Task Management** - Drag, drop, and click to manage tasks
- ✨ **Rich Dialogs** - Beautiful validation previews and commit flows

**Launch:**

```bash
# Start the GUI
ws queue-gui

# The GUI provides:
# - Main window with task list and status indicators
# - Toolbar with add, play, delete, refresh buttons
# - Dream button (🔍) for AI feature suggestions
# - macOS menu bar for quick actions
```

**Menu Bar (macOS):**
- **Show Queue** - Open the main window
- **Process Next Task** - Start next pending task
- **Add Task** - Quick add dialog
- **✨ Dream Feature** - AI suggests innovative features
- **Quit** - Exit application

**Dream Feature:**

The Dream button uses AI to analyze your codebase and propose innovative features:

1. Click the Dream button (🔍) in the toolbar
2. AI deeply explores your codebase using Glob and Grep patterns
3. AI reads key files and understands your architecture
4. AI proposes a specific, valuable feature with:
   - Feature name and description
   - Rationale for why it's valuable
   - Implementation approach
5. Add the suggestion directly as a task to your queue!

**Example Dream Workflow:**

```bash
ws queue-gui

# Click Dream button
# AI analyzes codebase for 30 seconds...

# Result:
# Feature: Workspace Templates
# Description: Pre-configured workspace setups for different task types
# Rationale: Eliminates repetitive configuration, ensures consistency
# Implementation: Add 'ws template' commands, store in ~/.config/ws/templates/

# Click "Add as Task" button
# → New task added to queue automatically!
```

**GUI Advantages:**
- **Visual feedback** - See all tasks at a glance with color-coded status
- **Menu bar access** - Quick actions without opening terminal
- **Rich interactions** - Dialogs for validation, commits, conflicts
- **AI integration** - Dream feature for creative suggestions
- **Background processing** - Tasks run while you work

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

### Environment Variables

Environment variables override config file settings:

```bash
WS_AGENT_CMD="claude --dangerously-skip-permissions"  # Agent for 'ws ez'
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

Fork me baby idgaf
