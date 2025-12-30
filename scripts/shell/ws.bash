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
