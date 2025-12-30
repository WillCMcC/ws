package main

import (
	"fmt"
	"os"

	"github.com/will/ws/cmd"
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
	case "list", "ls":
		exitCode = cmd.ListCmd(args)
	case "go":
		exitCode = cmd.GoCmd(args)
	case "done", "rm", "remove":
		exitCode = cmd.DoneCmd(args)
	case "status", "st":
		exitCode = cmd.StatusCmd(args)
	case "prune":
		exitCode = cmd.PruneCmd(args)
	case "init":
		exitCode = cmd.InitCmd(args)
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
  new <name>     Create a new workspace (worktree + branch)
  list           List all workspaces
  go <name>      Navigate to a workspace (requires shell integration)
  done <name>    Remove a workspace
  status         Show detailed status of all workspaces
  prune          Clean up stale worktrees
  init           Set up shell integration

Aliases:
  ls             Alias for 'list'
  rm, remove     Alias for 'done'
  st             Alias for 'status'

Examples:
  ws new auth-feature          Create workspace 'auth-feature'
  ws new bugfix --from develop Create from 'develop' branch
  ws list                      List all workspaces
  ws go auth-feature           Navigate to workspace
  ws done auth-feature         Remove workspace when done
  ws init                      Set up shell integration

Run 'ws <command> --help' for more information on a command.`)
}
