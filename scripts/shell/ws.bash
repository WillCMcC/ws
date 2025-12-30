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
            command ws go "$2"
            return $exit_code
        fi
    elif [[ "$1" == "home" ]]; then
        local target
        target=$(command ws home 2>/dev/null)
        local exit_code=$?
        if [[ $exit_code -eq 0 && -n "$target" && -d "$target" ]]; then
            cd "$target" || return 1
        else
            command ws home
            return $exit_code
        fi
    elif [[ "$1" == "new" && -n "$2" ]]; then
        command ws "$@"
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            local target
            target=$(command ws go "$2" 2>/dev/null)
            if [[ -n "$target" && -d "$target" ]]; then
                cd "$target" || return 1
            fi
        fi
        return $exit_code
    elif [[ "$1" == "ez" && -n "$2" ]]; then
        command ws "$@"
        local exit_code=$?
        if [[ $exit_code -eq 0 ]]; then
            local target
            target=$(command ws go "$2" 2>/dev/null)
            if [[ -n "$target" && -d "$target" ]]; then
                cd "$target" || return 1
                local agent_cmd
                agent_cmd=$(command ws agent-cmd)
                eval "$agent_cmd"
            fi
        fi
        return $exit_code
    else
        command ws "$@"
    fi
}

# Optional: completion
_ws_completions() {
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=($(compgen -W "new ez list go home done fold status prune init" -- "${COMP_WORDS[1]}"))
    elif [[ ${COMP_CWORD} -eq 2 ]]; then
        case "${COMP_WORDS[1]}" in
            go|done|fold|status)
                local workspaces
                workspaces=$(command ws list --quiet 2>/dev/null)
                COMPREPLY=($(compgen -W "$workspaces" -- "${COMP_WORDS[2]}"))
                ;;
        esac
    fi
}
complete -F _ws_completions ws
