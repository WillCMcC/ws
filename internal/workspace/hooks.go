package workspace

import (
	"os"
	"os/exec"
	"strings"
)

// runHook executes a hook command in the specified directory.
func runHook(command, dir, name string) error {
	// Replace placeholders
	command = strings.ReplaceAll(command, "{name}", name)
	command = strings.ReplaceAll(command, "{path}", dir)

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
