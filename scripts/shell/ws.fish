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
