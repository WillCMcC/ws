package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/will/ws/internal/workspace"
)

// ListCmd handles the 'ws list' command.
func ListCmd(args []string) int {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Just names, one per line")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ws list [--json] [--quiet]\n\n")
		fmt.Fprintf(os.Stderr, "List all workspaces for the current repository.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: %v\n", err)
		if err.Error() == "not a git repository" {
			fmt.Fprintf(os.Stderr, "    Run this command from within a git repository.\n")
			return 2
		}
		return 1
	}

	workspaces, err := mgr.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to list workspaces: %v\n", err)
		return 1
	}

	if *jsonOutput {
		return outputJSON(workspaces)
	}

	if *quiet {
		return outputQuiet(workspaces)
	}

	return outputTable(mgr, workspaces)
}

func outputJSON(workspaces []workspace.Workspace) int {
	type wsJSON struct {
		Name   string `json:"name"`
		Branch string `json:"branch"`
		Path   string `json:"path"`
	}

	var output []wsJSON
	for _, ws := range workspaces {
		output = append(output, wsJSON{
			Name:   ws.Name,
			Branch: ws.Branch,
			Path:   ws.Path,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to encode JSON: %v\n", err)
		return 1
	}
	return 0
}

func outputQuiet(workspaces []workspace.Workspace) int {
	for _, ws := range workspaces {
		fmt.Println(ws.Name)
	}
	return 0
}

func outputTable(mgr *workspace.Manager, workspaces []workspace.Workspace) int {
	if len(workspaces) == 0 {
		fmt.Println("No workspaces found.")
		fmt.Println()
		fmt.Println("Create one with: ws new <name>")
		return 0
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "WORKSPACE\tBRANCH\tPATH\tSTATUS")

	for _, ws := range workspaces {
		status := "idle"
		// Could add process detection here in the future
		path := shortenPath(ws.Path)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ws.Name, ws.Branch, path, status)
	}
	w.Flush()

	// Show main worktree info
	mainPath, mainBranch, _ := mgr.GetMainWorktree()
	fmt.Println()
	fmt.Printf("Main worktree: %s (branch: %s)\n", shortenPath(mainPath), mainBranch)

	return 0
}

// shortenPath replaces home directory with ~
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
