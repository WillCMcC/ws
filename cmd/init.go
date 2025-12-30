package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var bashIntegration = `# ws - workspace manager
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
complete -F _ws_completions ws`

var zshIntegration = `# ws - workspace manager
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
compdef _ws ws`

var fishIntegration = `# ws - workspace manager
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

complete -c ws -n "__fish_seen_subcommand_from go done" -a "(command ws list --quiet 2>/dev/null)"`

// InitCmd handles the 'ws init' command.
func InitCmd(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	shellType := fs.String("shell", "", "Shell type (bash, zsh, fish)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws init [--shell <type>]\n\n")
		fmt.Fprintf(os.Stderr, "Set up shell integration for ws.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	shell := *shellType
	if shell == "" {
		shell = detectShell()
	}

	var integration string
	var rcFile string

	home, _ := os.UserHomeDir()

	switch shell {
	case "bash":
		integration = bashIntegration
		rcFile = filepath.Join(home, ".bashrc")
	case "zsh":
		integration = zshIntegration
		rcFile = filepath.Join(home, ".zshrc")
	case "fish":
		integration = fishIntegration
		rcFile = filepath.Join(home, ".config", "fish", "conf.d", "ws.fish")
	default:
		fmt.Fprintf(os.Stderr, "ws: unknown shell '%s'\n", shell)
		fmt.Fprintf(os.Stderr, "    Supported shells: bash, zsh, fish\n")
		fmt.Fprintf(os.Stderr, "    Use --shell to specify: ws init --shell bash\n")
		return 1
	}

	fmt.Printf("Shell detected: %s\n", shell)
	fmt.Println()
	fmt.Printf("Add this to your %s:\n", rcFile)
	fmt.Println()
	fmt.Println(integration)
	fmt.Println()

	fmt.Print("Add automatically? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("To add manually, copy the code above to your shell config.")
		return 0
	}

	// Ensure directory exists for fish
	if shell == "fish" {
		dir := filepath.Dir(rcFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "ws: failed to create directory %s: %v\n", dir, err)
			return 1
		}
	}

	// Append to rc file
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to open %s: %v\n", rcFile, err)
		return 1
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + integration + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to write to %s: %v\n", rcFile, err)
		return 1
	}

	fmt.Printf("Added to %s\n", rcFile)
	fmt.Println()
	fmt.Println("Restart your shell or run:")
	fmt.Printf("  source %s\n", rcFile)

	return 0
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	if strings.Contains(shell, "bash") {
		return "bash"
	}
	// Default to bash
	return "bash"
}
