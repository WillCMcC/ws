package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/WillCMcC/ws/internal/git"
	"github.com/WillCMcC/ws/internal/queue"
	"github.com/WillCMcC/ws/internal/workspace"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	queueTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	statusStyle = map[queue.TaskStatus]lipgloss.Style{
		queue.StatusQueued: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		queue.StatusRunning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true),
		queue.StatusValidating: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		queue.StatusConflict: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		queue.StatusCompleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")),
		queue.StatusFailed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("160")),
	}

	queueSelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	queueHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	queueInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))
)

type viewMode int

const (
	viewQueue viewMode = iota
	viewAddTask
	viewTaskDetail
	viewValidation
	viewConflict
	viewCommitMessage
)

type queueModel struct {
	queue          *queue.Queue
	workspace      *workspace.Manager
	cursor         int
	mode           viewMode
	input          string
	taskNameInput  string
	message        string
	diffOutput     string
	conflictFiles  []string
	quitting       bool
	processing     bool
}

func RunQueue() int {
	q, err := queue.NewQueue()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to load queue: %v\n", err)
		return 1
	}

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws: failed to initialize workspace manager: %v\n", err)
		return 1
	}

	model := queueModel{
		queue:     q,
		workspace: mgr,
		mode:      viewQueue,
	}

	p := tea.NewProgram(&model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ws: error running queue UI: %v\n", err)
		return 1
	}

	return 0
}

func (m *queueModel) Init() tea.Cmd {
	return nil
}

func (m *queueModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case viewQueue:
			return m.updateQueueView(msg)
		case viewAddTask:
			return m.updateAddTaskView(msg)
		case viewTaskDetail:
			return m.updateTaskDetailView(msg)
		case viewValidation:
			return m.updateValidationView(msg)
		case viewCommitMessage:
			return m.updateCommitMessageView(msg)
		case viewConflict:
			return m.updateConflictView(msg)
		}
	}
	return m, nil
}

func (m *queueModel) updateQueueView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.queue.Tasks)-1 {
			m.cursor++
		}

	case "a":
		// Add new task
		m.mode = viewAddTask
		m.input = ""
		m.taskNameInput = ""
		m.message = ""

	case "enter":
		// View task details
		if len(m.queue.Tasks) > 0 && m.cursor < len(m.queue.Tasks) {
			m.mode = viewTaskDetail
		}

	case "n":
		// Process next task
		return m, m.processNextTask

	case "c":
		// Clear completed tasks
		if err := m.queue.Clear(); err != nil {
			m.message = fmt.Sprintf("Error clearing tasks: %v", err)
		} else {
			m.message = "Cleared completed tasks"
			m.cursor = 0
		}

	case "d":
		// Delete selected task
		if len(m.queue.Tasks) > 0 && m.cursor < len(m.queue.Tasks) {
			task := &m.queue.Tasks[m.cursor]
			if err := m.queue.Remove(task.ID); err != nil {
				m.message = fmt.Sprintf("Error removing task: %v", err)
			} else {
				m.message = fmt.Sprintf("Removed task: %s", task.Name)
				if m.cursor >= len(m.queue.Tasks) && m.cursor > 0 {
					m.cursor--
				}
			}
		}
	}

	return m, nil
}

func (m *queueModel) updateAddTaskView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewQueue
		m.input = ""
		m.taskNameInput = ""
		return m, nil

	case "enter":
		// First enter: save task name, ask for description
		if m.taskNameInput == "" {
			m.taskNameInput = m.input
			m.input = ""
			return m, nil
		}

		// Second enter: create task
		if m.taskNameInput != "" {
			task, err := m.queue.Add(m.taskNameInput, m.input)
			if err != nil {
				m.message = fmt.Sprintf("Error adding task: %v", err)
			} else {
				m.message = fmt.Sprintf("Added task: %s", task.Name)
			}
			m.mode = viewQueue
			m.input = ""
			m.taskNameInput = ""
		}
		return m, nil

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	default:
		// Accept printable characters
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}

	return m, nil
}

func (m *queueModel) updateTaskDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = viewQueue
		return m, nil

	case "s":
		// Start task
		if m.cursor < len(m.queue.Tasks) {
			task := &m.queue.Tasks[m.cursor]
			if task.IsPending() {
				return m, m.startTask(task)
			}
		}

	case "v":
		// Validate task
		if m.cursor < len(m.queue.Tasks) {
			task := &m.queue.Tasks[m.cursor]
			if task.Status == queue.StatusRunning {
				return m, m.validateTask(task)
			}
		}

	case "f":
		// Fold (complete) task
		if m.cursor < len(m.queue.Tasks) {
			task := &m.queue.Tasks[m.cursor]
			if task.Status == queue.StatusValidating {
				return m, m.foldTask(task)
			}
		}
	}

	return m, nil
}

func (m *queueModel) updateValidationView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewTaskDetail
		return m, nil

	case "y":
		// Accept changes and commit
		m.mode = viewCommitMessage
		m.input = ""
		return m, nil

	case "n":
		// Reject changes
		m.mode = viewTaskDetail
		m.message = "Changes rejected. Task remains in running state."
		return m, nil
	}

	return m, nil
}

func (m *queueModel) updateCommitMessageView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewValidation
		m.input = ""
		return m, nil

	case "enter":
		if m.input == "" {
			m.message = "Commit message cannot be empty"
			return m, nil
		}

		// Commit and fold
		task := &m.queue.Tasks[m.cursor]
		return m, m.commitAndFold(task, m.input)

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}

	return m, nil
}

func (m *queueModel) updateConflictView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = viewTaskDetail
		return m, nil

	case "a":
		// Run auto-rebase (launch agent to help resolve)
		task := &m.queue.Tasks[m.cursor]
		return m, m.runAutoRebase(task)

	case "r":
		// Retry fold after manual conflict resolution
		task := &m.queue.Tasks[m.cursor]
		return m, m.foldTask(task)
	}

	return m, nil
}

func (m *queueModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case viewQueue:
		return m.viewQueue()
	case viewAddTask:
		return m.viewAddTask()
	case viewTaskDetail:
		return m.viewTaskDetail()
	case viewValidation:
		return m.viewValidation()
	case viewCommitMessage:
		return m.viewCommitMessage()
	case viewConflict:
		return m.viewConflict()
	}

	return ""
}

func (m *queueModel) viewQueue() string {
	s := queueTitleStyle.Render("WS Task Queue") + "\n\n"

	if len(m.queue.Tasks) == 0 {
		s += queueHelpStyle.Render("No tasks in queue. Press 'a' to add a task.") + "\n"
	} else {
		for i, task := range m.queue.Tasks {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}

			status := statusStyle[task.Status].Render(string(task.Status))
			line := fmt.Sprintf("%s [%s] %s - %s", cursor, task.ID, task.Name, status)

			if i == m.cursor {
				line = queueSelectedStyle.Render(line)
			}

			s += line + "\n"
		}
	}

	if m.message != "" {
		s += "\n" + queueInputStyle.Render(m.message) + "\n"
	}

	s += "\n" + queueHelpStyle.Render("↑/k up • ↓/j down • enter detail • a add • n next • c clear • d delete • q quit")

	return s
}

func (m *queueModel) viewAddTask() string {
	s := queueTitleStyle.Render("Add New Task") + "\n\n"

	if m.taskNameInput == "" {
		s += "Task name: " + queueInputStyle.Render(m.input) + "█\n"
	} else {
		s += "Task name: " + m.taskNameInput + "\n"
		s += "Description: " + queueInputStyle.Render(m.input) + "█\n"
	}

	s += "\n" + queueHelpStyle.Render("enter confirm • esc cancel")

	return s
}

func (m *queueModel) viewTaskDetail() string {
	if m.cursor >= len(m.queue.Tasks) {
		return "No task selected"
	}

	task := &m.queue.Tasks[m.cursor]
	s := queueTitleStyle.Render(fmt.Sprintf("Task: %s", task.Name)) + "\n\n"

	s += fmt.Sprintf("ID:          %s\n", task.ID)
	s += fmt.Sprintf("Status:      %s\n", statusStyle[task.Status].Render(string(task.Status)))
	s += fmt.Sprintf("Description: %s\n", task.Description)
	s += fmt.Sprintf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

	if task.StartedAt != nil {
		s += fmt.Sprintf("Started:     %s\n", task.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if task.CompletedAt != nil {
		s += fmt.Sprintf("Completed:   %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if task.Error != "" {
		s += fmt.Sprintf("Error:       %s\n", task.Error)
	}

	if m.message != "" {
		s += "\n" + queueInputStyle.Render(m.message) + "\n"
	}

	s += "\n"
	if task.IsPending() {
		s += queueHelpStyle.Render("s start • esc back")
	} else if task.Status == queue.StatusRunning {
		s += queueHelpStyle.Render("v validate • esc back")
	} else if task.Status == queue.StatusValidating {
		s += queueHelpStyle.Render("f fold • esc back")
	} else if task.Status == queue.StatusConflict {
		s += queueHelpStyle.Render("Press 'esc' to return and view conflict resolution")
		m.mode = viewConflict
	} else {
		s += queueHelpStyle.Render("esc back")
	}

	return s
}

func (m *queueModel) viewValidation() string {
	task := &m.queue.Tasks[m.cursor]
	s := queueTitleStyle.Render(fmt.Sprintf("Validate: %s", task.Name)) + "\n\n"

	s += "Git changes:\n\n"
	s += m.diffOutput + "\n\n"

	s += queueHelpStyle.Render("y accept & commit • n reject • esc back")

	return s
}

func (m *queueModel) viewCommitMessage() string {
	task := &m.queue.Tasks[m.cursor]
	s := queueTitleStyle.Render(fmt.Sprintf("Commit: %s", task.Name)) + "\n\n"

	s += "Commit message: " + queueInputStyle.Render(m.input) + "█\n\n"

	s += queueHelpStyle.Render("enter commit & fold • esc back")

	return s
}

func (m *queueModel) viewConflict() string {
	task := &m.queue.Tasks[m.cursor]
	s := queueTitleStyle.Render(fmt.Sprintf("Conflict: %s", task.Name)) + "\n\n"

	s += statusStyle[queue.StatusConflict].Render("Merge conflicts detected during fold!") + "\n\n"

	s += "Conflicted files:\n"
	for _, file := range m.conflictFiles {
		s += "  - " + file + "\n"
	}

	s += "\n" + queueHelpStyle.Render("a run auto-rebase (launch agent) • r retry fold • esc back")

	return s
}

// Commands

func (m *queueModel) processNextTask() tea.Msg {
	next := m.queue.GetNextPending()
	if next == nil {
		m.message = "No pending tasks"
		return nil
	}

	return m.startTask(next)()
}

func (m *queueModel) startTask(task *queue.Task) tea.Cmd {
	return func() tea.Msg {
		// Update task status
		now := time.Now()
		task.Status = queue.StatusRunning
		task.StartedAt = &now
		if err := m.queue.UpdateTask(task); err != nil {
			m.message = fmt.Sprintf("Error updating task: %v", err)
			return nil
		}

		// Run ws ez <taskname>
		cmd := exec.Command("ws", "ez", task.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			task.Status = queue.StatusFailed
			task.Error = fmt.Sprintf("Failed to start workspace: %v", err)
			m.queue.UpdateTask(task)
			m.message = task.Error
			return nil
		}

		m.message = fmt.Sprintf("Started task: %s (workspace created)", task.Name)
		m.mode = viewTaskDetail
		return nil
	}
}

func (m *queueModel) validateTask(task *queue.Task) tea.Cmd {
	return func() tea.Msg {
		// Get workspace path
		wsDir := m.workspace.Config.GetWorkspaceDir(m.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, task.Name)

		// Check for uncommitted changes
		hasChanges, _, err := git.HasUncommittedChanges(wsPath)
		if err != nil {
			m.message = fmt.Sprintf("Error checking git status: %v", err)
			return nil
		}

		if !hasChanges {
			m.message = "No changes to validate"
			return nil
		}

		// Get diff output
		cmd := exec.Command("git", "-C", wsPath, "status", "--short")
		output, err := cmd.CombinedOutput()
		if err != nil {
			m.message = fmt.Sprintf("Error getting git status: %v", err)
			return nil
		}

		m.diffOutput = string(output)

		// Update task status
		task.Status = queue.StatusValidating
		m.queue.UpdateTask(task)

		m.mode = viewValidation
		return nil
	}
}

func (m *queueModel) commitAndFold(task *queue.Task, message string) tea.Cmd {
	return func() tea.Msg {
		// Get workspace path
		wsDir := m.workspace.Config.GetWorkspaceDir(m.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, task.Name)

		// Git add
		cmd := exec.Command("git", "-C", wsPath, "add", "-A")
		if err := cmd.Run(); err != nil {
			m.message = fmt.Sprintf("Error staging changes: %v", err)
			return nil
		}

		// Git commit
		cmd = exec.Command("git", "-C", wsPath, "commit", "-m", message)
		if err := cmd.Run(); err != nil {
			m.message = fmt.Sprintf("Error committing: %v", err)
			return nil
		}

		m.message = "Committed changes. Starting fold..."

		// Fold
		return m.foldTask(task)()
	}
}

func (m *queueModel) foldTask(task *queue.Task) tea.Cmd {
	return func() tea.Msg {
		// Run ws fold <taskname>
		cmd := exec.Command("ws", "fold", task.Name)
		output, err := cmd.CombinedOutput()

		if err != nil {
			// Check if it's a rebase conflict
			if strings.Contains(string(output), "conflict") ||
				strings.Contains(string(output), "CONFLICT") {
				task.Status = queue.StatusConflict
				m.queue.UpdateTask(task)

				// Get conflicted files
				m.conflictFiles = m.getConflictedFiles(task.Name)
				m.mode = viewConflict
				return nil
			}

			task.Status = queue.StatusFailed
			task.Error = fmt.Sprintf("Fold failed: %v\n%s", err, string(output))
			m.queue.UpdateTask(task)
			m.message = task.Error
			return nil
		}

		// Success!
		now := time.Now()
		task.Status = queue.StatusCompleted
		task.CompletedAt = &now
		m.queue.UpdateTask(task)

		m.message = fmt.Sprintf("Task completed: %s", task.Name)
		m.mode = viewTaskDetail
		return nil
	}
}

func (m *queueModel) runAutoRebase(task *queue.Task) tea.Cmd {
	return func() tea.Msg {
		// Get workspace path
		wsDir := m.workspace.Config.GetWorkspaceDir(m.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, task.Name)

		// Run ws auto-rebase
		cmd := exec.Command("ws", "auto-rebase")
		cmd.Dir = wsPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			m.message = fmt.Sprintf("Auto-rebase failed: %v", err)
			return nil
		}

		m.message = "Auto-rebase completed. Resolve conflicts manually, then press 'r' to retry fold."
		m.mode = viewConflict
		return nil
	}
}

func (m *queueModel) getConflictedFiles(taskName string) []string {
	wsDir := m.workspace.Config.GetWorkspaceDir(m.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, taskName)
	cmd := exec.Command("git", "-C", wsPath, "diff", "--name-only", "--diff-filter=U")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files
}
