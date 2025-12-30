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
