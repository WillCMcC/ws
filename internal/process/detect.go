package process

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// AgentProcess represents a detected agent process.
type AgentProcess struct {
	Name string
	PID  int
}

// DetectAgents finds agent processes running in a directory.
// It uses lsof to find processes with open files in the directory.
func DetectAgents(dir string, agentNames []string) []AgentProcess {
	var agents []AgentProcess

	// Use lsof to find processes with cwd in the directory
	// lsof +D is too slow, so we use a different approach:
	// Find processes and check their cwd
	for _, name := range agentNames {
		pids := findProcessesByName(name)
		for _, pid := range pids {
			cwd := getProcessCwd(pid)
			if cwd != "" && isInDirectory(cwd, dir) {
				agents = append(agents, AgentProcess{
					Name: name,
					PID:  pid,
				})
			}
		}
	}

	return agents
}

// findProcessesByName returns PIDs of processes matching the name.
func findProcessesByName(name string) []int {
	// Use pgrep to find processes by name
	cmd := exec.Command("pgrep", "-x", name)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		if pid, err := strconv.Atoi(line); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

// getProcessCwd returns the current working directory of a process.
func getProcessCwd(pid int) string {
	// On macOS, use lsof to get cwd
	// lsof -a -p <pid> -d cwd -Fn
	cmd := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse lsof output - look for line starting with 'n' (name field)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n")
		}
	}
	return ""
}

// isInDirectory checks if path is inside or equal to dir.
func isInDirectory(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	// Check if path equals dir or is inside it
	if absPath == absDir {
		return true
	}

	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
