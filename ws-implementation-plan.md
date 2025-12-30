# ws: Workspace Manager for Parallel Agent Development

## Overview

`ws` is a minimal CLI tool that manages git worktrees for running multiple coding agents in parallel on the same codebase. It's agent-agnostic, works with any CLI tool (Claude Code, OpenCode, Aider, Codex, Gemini CLI), and leaves all git operations (commit, merge, push) to the human.

## Design Principles

1. **Worktrees are workspaces** — each gets its own directory, branch, and isolated filesystem
2. **Agents are just processes** — ws doesn't know or care what runs in a workspace
3. **Git stays yours** — ws creates branches/worktrees; you commit, push, merge
4. **No daemon** — stateless CLI that reads from git/filesystem each invocation
5. **Shell integration for navigation** — `ws go <name>` changes directory via shell function

---

## Technical Specification

### Language & Distribution

- **Language:** Go 1.21+
- **Distribution:** Single static binary, no dependencies
- **Platforms:** Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- **Install methods:** 
  - curl one-liner (download binary)
  - `go install github.com/USER/ws@latest`
  - Homebrew tap (future)

### Project Structure

```
ws/
├── main.go                 # Entry point, command routing
├── cmd/
│   ├── new.go              # ws new
│   ├── list.go             # ws list
│   ├── go.go               # ws go (prints path for shell)
│   ├── done.go             # ws done
│   ├── status.go           # ws status
│   ├── prune.go            # ws prune
│   └── init.go             # ws init (shell setup helper)
├── internal/
│   ├── git/
│   │   ├── worktree.go     # Git worktree operations
│   │   ├── repo.go         # Repository detection/info
│   │   └── branch.go       # Branch utilities
│   ├── config/
│   │   ├── config.go       # Configuration loading
│   │   └── defaults.go     # Default values
│   ├── workspace/
│   │   ├── workspace.go    # Workspace model
│   │   ├── manager.go      # High-level workspace operations
│   │   └── hooks.go        # Pre/post hooks execution
│   └── shell/
│       ├── integration.go  # Shell script generation
│       └── detect.go       # Shell detection
├── scripts/
│   ├── install.sh          # Curl installer
│   └── shell/
│       ├── ws.bash         # Bash integration
│       ├── ws.zsh          # Zsh integration
│       └── ws.fish         # Fish integration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Commands

### `ws new <name> [--from <ref>] [--no-hooks]`

Create a new workspace (worktree + branch).

**Arguments:**
- `name` (required): Workspace name. Becomes the branch name and directory name.

**Flags:**
- `--from <ref>`: Base ref to branch from. Default: `main` (configurable)
- `--no-hooks`: Skip post-create hooks

**Behavior:**
1. Validate we're in a git repository
2. Validate `name` doesn't already exist as worktree or branch
3. Determine worktree directory path (see Directory Layout below)
4. Run `git worktree add -b <name> <path> <from>`
5. If hooks configured, run post-create hook in new worktree directory
6. Print success message with path and suggested next command

**Output:**
```
Created workspace: auth-feature
  Path:   /home/user/myproject-ws/auth-feature
  Branch: auth-feature (from main)

  cd /home/user/myproject-ws/auth-feature && claude
  
  Or use: ws go auth-feature
```

**Errors:**
- Not in a git repository → exit 1 with message
- Name already exists → exit 1 with message
- Git worktree add fails → exit 1, show git error
- Hook fails → warn but don't fail (workspace was created)

---

### `ws list`

List all workspaces for current repository.

**Behavior:**
1. Find repository root
2. Run `git worktree list --porcelain`
3. Parse output, filter to workspaces we manage (in configured workspace directory)
4. For each, gather: name, path, branch, last modified time
5. Optionally detect running processes (see Status Detection)

**Output (table format):**
```
WORKSPACE       BRANCH          PATH                                    STATUS
auth-feature    auth-feature    ~/myproject-ws/auth-feature             active
payment-fix     payment-fix     ~/myproject-ws/payment-fix              idle
api-refactor    api-refactor    ~/myproject-ws/api-refactor             idle

Main worktree: ~/myproject (branch: main)
```

**Flags:**
- `--json`: Output as JSON array
- `--quiet`: Just names, one per line

---

### `ws go <name>`

Navigate to a workspace directory.

**Important:** This command cannot actually change the shell's directory. It prints the path, and shell integration handles the `cd`.

**Behavior:**
1. Find workspace by name
2. Verify it exists
3. Print absolute path to stdout (nothing else)

**Shell Integration Required:**
```bash
# In .bashrc/.zshrc
ws() {
    if [[ "$1" == "go" && -n "$2" ]]; then
        local target
        target=$(command ws go "$2" 2>/dev/null)
        if [[ -n "$target" && -d "$target" ]]; then
            cd "$target"
        else
            command ws go "$2"  # Let it print error
        fi
    else
        command ws "$@"
    fi
}
```

**Errors:**
- Workspace not found → exit 1 with message

---

### `ws done <name> [--force]`

Remove a workspace (worktree + optionally branch).

**Arguments:**
- `name` (required): Workspace to remove

**Flags:**
- `--force`: Remove even with uncommitted changes
- `--keep-branch`: Don't delete the branch (default: delete branch)

**Behavior:**
1. Find workspace by name
2. Check for uncommitted changes in worktree
3. If uncommitted changes and no --force → error with instructions
4. Run `git worktree remove <path>` (or with --force)
5. Unless --keep-branch, run `git branch -d <name>` (or -D if --force)
6. Print success

**Output:**
```
Removed workspace: auth-feature
  Branch auth-feature deleted
```

**With uncommitted changes:**
```
Workspace auth-feature has uncommitted changes:
  M src/auth.ts
  A src/login.ts

To remove anyway: ws done auth-feature --force
To keep changes:  cd ~/myproject-ws/auth-feature && git stash
```

---

### `ws status`

Show detailed status of all workspaces.

**Behavior:**
1. List all workspaces
2. For each workspace:
   - Check git status (clean/dirty, ahead/behind)
   - Detect running processes (optional, best-effort)
   - Get last commit info

**Output:**
```
auth-feature
  Path:     ~/myproject-ws/auth-feature
  Branch:   auth-feature (3 commits ahead of main)
  Status:   2 modified, 1 untracked
  Last:     "Add login form" (2 hours ago)
  Process:  claude (pid 12345)

payment-fix
  Path:     ~/myproject-ws/payment-fix
  Branch:   payment-fix (1 commit ahead of main)
  Status:   clean
  Last:     "Fix payment validation" (1 day ago)
  Process:  none detected
```

**Process Detection (best-effort):**
- Look for known agent processes whose cwd is in the workspace
- Check for: `claude`, `opencode`, `aider`, `codex`, `gemini`
- Use `lsof` or `/proc` on Linux, `lsof` on macOS
- Fail silently if detection doesn't work

---

### `ws prune`

Clean up stale worktrees.

**Behavior:**
1. Run `git worktree prune` (cleans git's internal state)
2. Find workspace directories that no longer have worktrees
3. Optionally remove orphaned directories

**Output:**
```
Pruned git worktree metadata

Found orphaned directories:
  ~/myproject-ws/old-feature (no worktree, last modified 7 days ago)

Remove orphaned directories? [y/N]
```

**Flags:**
- `--dry-run`: Show what would be removed
- `--yes`: Don't prompt for orphan removal

---

### `ws init [--shell <type>]`

Help user set up shell integration.

**Behavior:**
1. Detect current shell (or use --shell flag)
2. Print instructions for adding shell integration
3. Optionally append to shell rc file (with confirmation)

**Output:**
```
Shell detected: zsh

Add this to your ~/.zshrc:

  # ws - workspace manager
  ws() {
      if [[ "$1" == "go" && -n "$2" ]]; then
          local target
          target=$(command ws go "$2" 2>/dev/null)
          if [[ -n "$target" && -d "$target" ]]; then
              cd "$target"
          else
              command ws go "$2"
          fi
      else
          command ws "$@"
      fi
  }

Add automatically? [y/N]
```

---

## Configuration

### Config File Location

1. `.ws.toml` in repository root (project-specific)
2. `~/.config/ws/config.toml` (user global)
3. Environment variables (override all)

Project config overrides user config.

### Config Schema

```toml
# .ws.toml or ~/.config/ws/config.toml

[workspace]
# Where to put worktrees, relative to repo root
# Supports {repo} placeholder for repo basename
# Default: "../{repo}-ws"
directory = "../{repo}-ws"

# Default base branch for new workspaces
# Default: "main"
default_base = "main"

[hooks]
# Command to run after creating workspace (in workspace directory)
# Supports {name}, {path}, {branch} placeholders
post_create = "npm install"

# Command to run before removing workspace
pre_remove = ""

[status]
# Try to detect running agent processes
# Default: true
detect_processes = true

# Known agent process names to look for
# Default: ["claude", "opencode", "aider", "codex", "gemini"]
agent_processes = ["claude", "opencode", "aider", "codex", "gemini"]
```

### Environment Variables

```bash
WS_DIRECTORY="./workspaces"      # Override workspace directory
WS_DEFAULT_BASE="develop"         # Override default base branch
WS_NO_HOOKS="1"                   # Disable all hooks
```

---

## Directory Layout

### Default Layout

Given a repo at `/home/user/projects/myapp`:

```
/home/user/projects/
├── myapp/                      # Main repository (main worktree)
│   ├── .git/
│   ├── .ws.toml                # Optional project config
│   └── src/
└── myapp-ws/                   # Workspace directory
    ├── auth-feature/           # Worktree for auth-feature branch
    │   ├── .git                # File pointing to main .git
    │   └── src/
    └── payment-fix/            # Worktree for payment-fix branch
        ├── .git
        └── src/
```

### Configurable via `workspace.directory`

Examples:
- `"../workspaces/{repo}"` → `/home/user/projects/workspaces/myapp/`
- `"./.workspaces"` → `/home/user/projects/myapp/.workspaces/`
- `"/tmp/ws/{repo}"` → `/tmp/ws/myapp/`

---

## Git Operations

All git operations shell out to `git`. Don't use a git library.

### Repository Detection

```go
func FindRepoRoot() (string, error) {
    cmd := exec.Command("git", "rev-parse", "--show-toplevel")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("not a git repository")
    }
    return strings.TrimSpace(string(output)), nil
}
```

### Worktree Operations

```go
func CreateWorktree(path, branch, base string) error {
    cmd := exec.Command("git", "worktree", "add", "-b", branch, path, base)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func RemoveWorktree(path string, force bool) error {
    args := []string{"worktree", "remove", path}
    if force {
        args = append(args, "--force")
    }
    cmd := exec.Command("git", args...)
    return cmd.Run()
}

func ListWorktrees() ([]Worktree, error) {
    cmd := exec.Command("git", "worktree", "list", "--porcelain")
    output, err := cmd.Output()
    // Parse porcelain format...
}
```

### Porcelain Format Parsing

`git worktree list --porcelain` outputs:
```
worktree /home/user/myapp
HEAD abc123...
branch refs/heads/main

worktree /home/user/myapp-ws/feature
HEAD def456...
branch refs/heads/feature
```

Parse by splitting on blank lines, then key-value pairs.

---

## Shell Integration Scripts

### Bash (scripts/shell/ws.bash)

```bash
# ws shell integration for bash
# Add to ~/.bashrc: source /path/to/ws.bash

ws() {
    if [[ "$1" == "go" && -n "$2" ]]; then
        local target
        target=$(command ws go "$2" 2>/dev/null)
        local exit_code=$?
        if [[ $exit_code -eq 0 && -n "$target" && -d "$target" ]]; then
            cd "$target" || return 1
        else
            # Re-run to show error message
            command ws go "$2"
            return $exit_code
        fi
    else
        command ws "$@"
    fi
}

# Optional: completion
_ws_completions() {
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=($(compgen -W "new list go done status prune init" -- "${COMP_WORDS[1]}"))
    elif [[ ${COMP_CWORD} -eq 2 ]]; then
        case "${COMP_WORDS[1]}" in
            go|done|status)
                local workspaces
                workspaces=$(command ws list --quiet 2>/dev/null)
                COMPREPLY=($(compgen -W "$workspaces" -- "${COMP_WORDS[2]}"))
                ;;
        esac
    fi
}
complete -F _ws_completions ws
```

### Zsh (scripts/shell/ws.zsh)

```zsh
# ws shell integration for zsh
# Add to ~/.zshrc: source /path/to/ws.zsh

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

# Completion
_ws() {
    local -a commands
    commands=(
        'new:Create a new workspace'
        'list:List all workspaces'
        'go:Navigate to a workspace'
        'done:Remove a workspace'
        'status:Show workspace status'
        'prune:Clean up stale worktrees'
        'init:Set up shell integration'
    )
    
    if (( CURRENT == 2 )); then
        _describe 'command' commands
    elif (( CURRENT == 3 )); then
        case "$words[2]" in
            go|done)
                local -a workspaces
                workspaces=(${(f)"$(command ws list --quiet 2>/dev/null)"})
                _describe 'workspace' workspaces
                ;;
        esac
    fi
}
compdef _ws ws
```

### Fish (scripts/shell/ws.fish)

```fish
# ws shell integration for fish
# Add to ~/.config/fish/conf.d/ws.fish

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

# Completion
complete -c ws -n "__fish_use_subcommand" -a new -d "Create a new workspace"
complete -c ws -n "__fish_use_subcommand" -a list -d "List all workspaces"
complete -c ws -n "__fish_use_subcommand" -a go -d "Navigate to a workspace"
complete -c ws -n "__fish_use_subcommand" -a done -d "Remove a workspace"
complete -c ws -n "__fish_use_subcommand" -a status -d "Show workspace status"
complete -c ws -n "__fish_use_subcommand" -a prune -d "Clean up stale worktrees"
complete -c ws -n "__fish_use_subcommand" -a init -d "Set up shell integration"

complete -c ws -n "__fish_seen_subcommand_from go done" -a "(command ws list --quiet 2>/dev/null)"
```

---

## Error Handling

### Exit Codes

- `0`: Success
- `1`: General error (invalid args, operation failed)
- `2`: Not in a git repository
- `3`: Workspace not found
- `4`: Workspace already exists

### Error Messages

Always prefix with `ws:` and be actionable:

```
ws: not in a git repository
    Run this command from within a git repository.

ws: workspace 'auth-feature' already exists
    Path: /home/user/myapp-ws/auth-feature
    To remove it: ws done auth-feature

ws: workspace 'foo' not found
    Available workspaces:
      auth-feature
      payment-fix
    To create it: ws new foo

ws: workspace 'auth-feature' has uncommitted changes
    To see changes: cd /home/user/myapp-ws/auth-feature && git status
    To remove anyway: ws done auth-feature --force
```

---

## Testing Strategy

### Unit Tests

- Config parsing
- Worktree list parsing
- Path template expansion
- Workspace name validation

### Integration Tests

Use a temporary directory with a real git repo:

```go
func TestNewWorkspace(t *testing.T) {
    // Create temp dir
    dir := t.TempDir()
    
    // Initialize git repo
    runGit(dir, "init")
    runGit(dir, "commit", "--allow-empty", "-m", "init")
    
    // Run ws new
    ws := NewManager(dir)
    err := ws.Create("test-feature", "main")
    
    // Assert worktree exists
    // Assert branch exists
    // Assert directory structure
}
```

### Test Cases

**ws new:**
- Creates worktree and branch
- Uses correct base branch
- Respects custom workspace directory
- Runs post-create hook
- Fails gracefully if branch exists
- Fails gracefully if not in git repo

**ws list:**
- Shows all workspaces
- Handles empty list
- JSON output is valid
- Quiet mode shows only names

**ws go:**
- Prints correct path
- Exits 3 if not found

**ws done:**
- Removes worktree
- Deletes branch
- Warns on uncommitted changes
- --force bypasses warning
- --keep-branch preserves branch

**ws status:**
- Shows git status per workspace
- Process detection works (when possible)
- Graceful degradation if detection fails

**ws prune:**
- Cleans git worktree metadata
- Identifies orphan directories
- Prompts before removing orphans

---

## Build & Release

### Makefile

```makefile
.PHONY: build test install clean release

VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o ws .

test:
	go test -v ./...

install: build
	mv ws /usr/local/bin/

clean:
	rm -f ws
	rm -rf dist/

release:
	goreleaser release --clean
```

### GoReleaser Config (.goreleaser.yaml)

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - main: .
    binary: ws
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE
      - scripts/shell/*

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc

release:
  github:
    owner: YOUR_USERNAME
    name: ws
```

### Install Script (scripts/install.sh)

```bash
#!/bin/bash
set -euo pipefail

VERSION="${1:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -s https://api.github.com/repos/USER/ws/releases/latest | grep tag_name | cut -d '"' -f 4)
fi

URL="https://github.com/USER/ws/releases/download/${VERSION}/ws_${OS}_${ARCH}.tar.gz"

echo "Downloading ws ${VERSION} for ${OS}/${ARCH}..."
curl -sL "$URL" | tar xz -C /tmp

echo "Installing to ${INSTALL_DIR}/ws..."
sudo mv /tmp/ws "$INSTALL_DIR/ws"
sudo chmod +x "$INSTALL_DIR/ws"

echo "Installed successfully!"
echo ""
echo "Run 'ws init' to set up shell integration."
```

---

## Implementation Order

Recommended order for building:

### Phase 1: Core (MVP)
1. `main.go` - arg parsing, command routing
2. `internal/git/repo.go` - repo detection
3. `internal/git/worktree.go` - worktree operations
4. `cmd/new.go` - create workspace
5. `cmd/list.go` - list workspaces
6. `cmd/go.go` - print path
7. `cmd/done.go` - remove workspace
8. Shell integration scripts

**Deliverable:** Working `ws new`, `ws list`, `ws go`, `ws done`

### Phase 2: Polish
1. `internal/config/config.go` - config file loading
2. `internal/workspace/hooks.go` - post-create hooks
3. `cmd/init.go` - shell setup helper
4. Tab completion in shell scripts
5. `--json` and `--quiet` flags

**Deliverable:** Configuration, hooks, better UX

### Phase 3: Extended Features
1. `cmd/status.go` - detailed status
2. Process detection
3. `cmd/prune.go` - cleanup
4. Error messages polish
5. Tests

**Deliverable:** Full feature set

### Phase 4: Distribution
1. GoReleaser setup
2. Install script
3. README with examples
4. GitHub Actions for CI/CD

---

## Example Session

```bash
# Setup (one-time)
curl -fsSL https://raw.githubusercontent.com/USER/ws/main/scripts/install.sh | bash
ws init  # adds shell integration

# Daily workflow
cd ~/projects/myapp

ws new auth-feature
# Created workspace: auth-feature
#   Path: /home/user/projects/myapp-ws/auth-feature
#   Branch: auth-feature (from main)

ws go auth-feature
# (now in /home/user/projects/myapp-ws/auth-feature)

claude  # or opencode, aider, etc.
# ... agent works on auth feature ...

# In another terminal
cd ~/projects/myapp
ws new payment-fix
ws go payment-fix
claude
# ... agent works on payment fix ...

# Check status
ws list
# WORKSPACE       BRANCH          STATUS
# auth-feature    auth-feature    active
# payment-fix     payment-fix     active

# When done with auth feature
cd ~/projects/myapp-ws/auth-feature
git add -A && git commit -m "Implement auth"
cd ~/projects/myapp
git merge auth-feature
ws done auth-feature
# Removed workspace: auth-feature
```

---

## Future Enhancements (Not MVP)

- **tmux integration:** `ws new --attach` opens new tmux window
- **Workspace templates:** Copy files (.env, etc.) from main worktree
- **Branch sync:** Show if branch is behind main
- **Auto-cleanup:** Remove workspaces older than N days
- **Watch mode:** `ws watch` shows live status updates
- **Git stash integration:** Stash changes before removing

---

## Notes for Implementing Agent

1. Start with Phase 1 (MVP) and get it working end-to-end before adding features
2. Shell out to `git` for all git operations - don't use go-git or similar
3. Keep the code simple; this is a ~500-1000 line tool, not a framework
4. Test manually with a real git repo frequently during development
5. The shell integration is critical - test it in bash, zsh, and fish
6. Error messages should be helpful and actionable
7. Prefer explicit over clever - this tool should be boring and reliable
