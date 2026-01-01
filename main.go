package main

import (
	"fmt"
	"os"

	"github.com/WillCMcC/ws/cmd"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]
	args := os.Args[2:]

	var exitCode int

	switch command {
	case "new":
		exitCode = cmd.NewCmd(args)
	case "ez":
		exitCode = cmd.EzCmd(args)
	case "list", "ls":
		exitCode = cmd.ListCmd(args)
	case "go":
		exitCode = cmd.GoCmd(args)
	case "home":
		exitCode = cmd.HomeCmd(args)
	case "done", "rm", "remove":
		exitCode = cmd.DoneCmd(args)
	case "fold":
		exitCode = cmd.FoldCmd(args)
	case "auto-rebase":
		exitCode = cmd.AutoRebaseCmd(args)
	case "status", "st":
		exitCode = cmd.StatusCmd(args)
	case "prune":
		exitCode = cmd.PruneCmd(args)
	case "init":
		exitCode = cmd.InitCmd(args)
	case "config":
		exitCode = cmd.ConfigCmd(args)
	case "queue":
		exitCode = cmd.RunQueue()
	case "queue-gui":
		exitCode = cmd.RunQueueGUI()
	case "agent-cmd":
		// Internal command for shell integration to get agent command
		fmt.Print(cmd.GetAgentCmd())
		exitCode = 0
	case "version", "--version", "-v":
		fmt.Printf("ws version %s\n", version)
		exitCode = 0
	case "help", "--help", "-h":
		printUsage()
		exitCode = 0
	default:
		fmt.Fprintf(os.Stderr, "ws: unknown command '%s'\n", command)
		fmt.Fprintf(os.Stderr, "    Run 'ws help' for usage.\n")
		exitCode = 1
	}

	os.Exit(exitCode)
}

func printUsage() {
	fmt.Println(`ws - Workspace manager for parallel agent development

Usage: ws <command> [arguments]

Commands:
  new <name>     Create a new workspace and navigate to it
  ez <name>      Create workspace, navigate, and start agent
  list           List all workspaces
  go <name>      Navigate to a workspace
  home           Navigate to main repository
  done <name>    Remove a workspace
  fold [name]    Rebase and merge workspace into default branch
  auto-rebase    Start agent to help resolve rebase conflicts
  status         Show detailed status of all workspaces
  prune          Clean up stale worktrees
  init           Set up shell integration
  config         Manage configuration
  queue          Interactive task queue manager (TUI)
  queue-gui      Graphical task queue manager with menu bar

Aliases:
  ls             Alias for 'list'
  rm, remove     Alias for 'done'
  st             Alias for 'status'

Examples:
  ws new auth-feature          Create workspace and cd into it
  ws ez auth-feature           Create, cd, and start agent
  ws list                      List all workspaces
  ws go auth-feature           Navigate to workspace
  ws home                      Navigate back to main repo
  ws fold                      Rebase and merge current workspace
  ws done auth-feature         Remove workspace when done

Environment:
  WS_AGENT_CMD   Agent command for 'ws ez' (default: claude)

Run 'ws <command> --help' for more information on a command.`)
}
