package cmd

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/WillCMcC/ws/internal/dream"
	"github.com/WillCMcC/ws/internal/git"
	"github.com/WillCMcC/ws/internal/queue"
	"github.com/WillCMcC/ws/internal/workspace"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/getlantern/systray"
)

// RunQueueGUI starts the GUI task queue manager
func RunQueueGUI() int {
	// Create Fyne app
	myApp := app.NewWithID("com.ws.queue")
	myApp.SetIcon(theme.DocumentIcon())

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

	// Start menu bar in background
	go runMenuBar(q, myApp)

	// Create main window
	win := myApp.NewWindow("WS Task Queue Manager")
	win.Resize(fyne.NewSize(900, 600))

	gui := &queueGUI{
		app:       myApp,
		window:    win,
		queue:     q,
		workspace: mgr,
	}

	gui.buildUI()

	win.ShowAndRun()
	return 0
}

type queueGUI struct {
	app       fyne.App
	window    fyne.Window
	queue     *queue.Queue
	workspace *workspace.Manager

	// UI components
	taskList   *widget.List
	statusBar  *widget.Label
	taskData   binding.StringList
}

func (g *queueGUI) buildUI() {
	// Create toolbar
	toolbar := g.createToolbar()

	// Create task list
	g.taskData = binding.NewStringList()
	g.refreshTaskList()

	g.taskList = widget.NewListWithData(
		g.taskData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Task"),
				layout.NewSpacer(),
				widget.NewLabel("Status"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			strItem := item.(binding.String)
			text, _ := strItem.Get()

			parts := strings.Split(text, " | ")
			if len(parts) >= 2 {
				box := obj.(*fyne.Container)
				box.Objects[0].(*widget.Label).SetText(parts[0])
				statusLabel := box.Objects[2].(*widget.Label)
				statusLabel.SetText(parts[1])

				// Color code status
				switch parts[1] {
				case "queued":
					statusLabel.Importance = widget.LowImportance
				case "running":
					statusLabel.Importance = widget.WarningImportance
				case "completed":
					statusLabel.Importance = widget.SuccessImportance
				case "failed", "conflict":
					statusLabel.Importance = widget.DangerImportance
				}
			}
		},
	)

	g.taskList.OnSelected = func(id widget.ListItemID) {
		g.showTaskDetails(id)
	}

	// Status bar
	g.statusBar = widget.NewLabel("Ready")
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true}

	// Main layout
	content := container.NewBorder(
		toolbar,
		g.statusBar,
		nil,
		nil,
		g.taskList,
	)

	g.window.SetContent(content)
}

func (g *queueGUI) createToolbar() *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			g.showAddTaskDialog()
		}),
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			g.processNextTask()
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			g.deleteSelectedTask()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			g.refreshTaskList()
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.SearchIcon(), func() {
			g.showDreamDialog()
		}),
	)
}

func (g *queueGUI) refreshTaskList() {
	items := []string{}
	for _, task := range g.queue.Tasks {
		items = append(items, fmt.Sprintf("%s | %s | %s",
			task.Name,
			task.Status,
			task.Description,
		))
	}
	g.taskData.Set(items)
}

func (g *queueGUI) showAddTaskDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Task name (e.g., auth-feature)")

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Task description")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: nameEntry},
			{Text: "Description", Widget: descEntry},
		},
		OnSubmit: func() {
			name := nameEntry.Text
			desc := descEntry.Text

			if name == "" {
				dialog.ShowError(fmt.Errorf("task name is required"), g.window)
				return
			}

			_, err := g.queue.Add(name, desc)
			if err != nil {
				dialog.ShowError(err, g.window)
				return
			}

			g.refreshTaskList()
			g.setStatus(fmt.Sprintf("Added task: %s", name))
		},
	}

	dialog.ShowForm("Add New Task", "Add", "Cancel", form.Items, form.OnSubmit, g.window)
}

func (g *queueGUI) processNextTask() {
	next := g.queue.GetNextPending()
	if next == nil {
		dialog.ShowInformation("No Tasks", "No pending tasks in queue", g.window)
		return
	}

	g.setStatus(fmt.Sprintf("Starting task: %s", next.Name))

	// Update task status
	now := time.Now()
	next.Status = queue.StatusRunning
	next.StartedAt = &now
	g.queue.UpdateTask(next)
	g.refreshTaskList()

	// Run ws ez in background
	go func() {
		cmd := exec.Command("ws", "ez", next.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			next.Status = queue.StatusFailed
			next.Error = fmt.Sprintf("Failed to start workspace: %v", err)
			g.queue.UpdateTask(next)
			g.refreshTaskList()
			g.setStatus(fmt.Sprintf("Task failed: %s", next.Name))

			dialog.ShowError(fmt.Errorf("failed to start task: %v", err), g.window)
			return
		}

		g.setStatus(fmt.Sprintf("Task running: %s", next.Name))
		g.refreshTaskList()
	}()
}

func (g *queueGUI) showTaskDetails(id widget.ListItemID) {
	if id >= len(g.queue.Tasks) {
		return
	}

	task := &g.queue.Tasks[id]

	details := fmt.Sprintf(`Task: %s
ID: %s
Status: %s
Description: %s
Created: %s`,
		task.Name,
		task.ID,
		task.Status,
		task.Description,
		task.CreatedAt.Format("2006-01-02 15:04:05"),
	)

	if task.StartedAt != nil {
		details += fmt.Sprintf("\nStarted: %s", task.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if task.CompletedAt != nil {
		details += fmt.Sprintf("\nCompleted: %s", task.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if task.Error != "" {
		details += fmt.Sprintf("\nError: %s", task.Error)
	}

	detailLabel := widget.NewLabel(details)
	detailLabel.Wrapping = fyne.TextWrapWord

	buttons := container.NewHBox()

	if task.IsPending() {
		buttons.Add(widget.NewButton("Start Task", func() {
			g.taskList.Select(id)
			g.processNextTask()
		}))
	} else if task.Status == queue.StatusRunning {
		buttons.Add(widget.NewButton("Validate & Commit", func() {
			g.showValidationDialog(task)
		}))
	} else if task.Status == queue.StatusConflict {
		buttons.Add(widget.NewButton("Run Auto-Rebase", func() {
			g.runAutoRebase(task)
		}))
		buttons.Add(widget.NewButton("Retry Fold", func() {
			g.foldTask(task)
		}))
	}

	content := container.NewBorder(
		detailLabel,
		buttons,
		nil,
		nil,
		nil,
	)

	d := dialog.NewCustom("Task Details", "Close", content, g.window)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

func (g *queueGUI) showValidationDialog(task *queue.Task) {
	wsDir := g.workspace.Config.GetWorkspaceDir(g.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, task.Name)

	// Check for changes
	hasChanges, _, err := git.HasUncommittedChanges(wsPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("error checking git status: %v", err), g.window)
		return
	}

	if !hasChanges {
		dialog.ShowInformation("No Changes", "No uncommitted changes in workspace", g.window)
		return
	}

	// Get git status
	cmd := exec.Command("git", "-C", wsPath, "status", "--short")
	output, _ := cmd.CombinedOutput()

	diffText := widget.NewMultiLineEntry()
	diffText.SetText(string(output))
	diffText.Disable()

	messageEntry := widget.NewEntry()
	messageEntry.SetPlaceHolder("Commit message")

	form := container.NewVBox(
		widget.NewLabel("Changes:"),
		diffText,
		widget.NewSeparator(),
		widget.NewLabel("Commit Message:"),
		messageEntry,
	)

	dialog.ShowCustomConfirm(
		"Validate Changes",
		"Commit & Fold",
		"Cancel",
		form,
		func(ok bool) {
			if !ok {
				return
			}

			if messageEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("commit message required"), g.window)
				return
			}

			g.commitAndFold(task, messageEntry.Text)
		},
		g.window,
	)
}

func (g *queueGUI) commitAndFold(task *queue.Task, message string) {
	wsDir := g.workspace.Config.GetWorkspaceDir(g.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, task.Name)

	// Git add
	cmd := exec.Command("git", "-C", wsPath, "add", "-A")
	if err := cmd.Run(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to stage changes: %v", err), g.window)
		return
	}

	// Git commit
	cmd = exec.Command("git", "-C", wsPath, "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to commit: %v", err), g.window)
		return
	}

	g.setStatus("Committed changes, starting fold...")

	// Fold
	go g.foldTask(task)
}

func (g *queueGUI) foldTask(task *queue.Task) {
	cmd := exec.Command("ws", "fold", task.Name)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check for conflicts
		if strings.Contains(string(output), "conflict") ||
			strings.Contains(string(output), "CONFLICT") {
			task.Status = queue.StatusConflict
			g.queue.UpdateTask(task)
			g.refreshTaskList()

			dialog.ShowError(fmt.Errorf("merge conflicts detected:\n%s", string(output)), g.window)
			return
		}

		task.Status = queue.StatusFailed
		task.Error = fmt.Sprintf("Fold failed: %v", err)
		g.queue.UpdateTask(task)
		g.refreshTaskList()

		dialog.ShowError(fmt.Errorf("fold failed: %v", err), g.window)
		return
	}

	// Success
	now := time.Now()
	task.Status = queue.StatusCompleted
	task.CompletedAt = &now
	g.queue.UpdateTask(task)
	g.refreshTaskList()

	g.setStatus(fmt.Sprintf("Task completed: %s", task.Name))
	dialog.ShowInformation("Success", fmt.Sprintf("Task '%s' completed and merged!", task.Name), g.window)
}

func (g *queueGUI) runAutoRebase(task *queue.Task) {
	wsDir := g.workspace.Config.GetWorkspaceDir(g.workspace.RepoRoot)
	wsPath := filepath.Join(wsDir, task.Name)

	g.setStatus("Running auto-rebase...")

	go func() {
		cmd := exec.Command("ws", "auto-rebase")
		cmd.Dir = wsPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			dialog.ShowError(fmt.Errorf("auto-rebase failed: %v", err), g.window)
			return
		}

		g.setStatus("Auto-rebase completed. Resolve conflicts and retry fold.")
		dialog.ShowInformation("Auto-Rebase", "Auto-rebase completed. Resolve conflicts manually, then retry fold.", g.window)
	}()
}

func (g *queueGUI) deleteSelectedTask() {
	selected := g.taskList.SelectedIndex()
	if selected < 0 || selected >= len(g.queue.Tasks) {
		dialog.ShowInformation("No Selection", "Please select a task to delete", g.window)
		return
	}

	task := &g.queue.Tasks[selected]

	dialog.ShowConfirm(
		"Delete Task",
		fmt.Sprintf("Are you sure you want to delete task '%s'?", task.Name),
		func(ok bool) {
			if ok {
				g.queue.Remove(task.ID)
				g.refreshTaskList()
				g.setStatus(fmt.Sprintf("Deleted task: %s", task.Name))
			}
		},
		g.window,
	)
}

func (g *queueGUI) showDreamDialog() {
	progress := widget.NewProgressBarInfinite()
	resultLabel := widget.NewLabel("Analyzing codebase and dreaming up features...")
	resultLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		widget.NewLabelWithStyle("🌟 Dream Feature Generator", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("The AI is deeply investigating your codebase to propose innovative features..."),
		progress,
		container.NewVScroll(resultLabel),
	)

	d := dialog.NewCustom("Dream Feature", "Close", content, g.window)
	d.Resize(fyne.NewSize(600, 500))
	d.Show()

	// Run dream analysis in background
	go func() {
		result := g.runDreamAnalysis()

		progress.Stop()
		progress.Hide()
		resultLabel.SetText(result)
		resultLabel.Refresh()

		// Ask if user wants to add as task
		if strings.Contains(result, "Feature:") {
			addButton := widget.NewButton("Add as Task", func() {
				g.addDreamAsTask(result)
				d.Hide()
			})
			content.Add(addButton)
		}
	}()
}

func (g *queueGUI) runDreamAnalysis() string {
	// Use the dream analyzer to perform real AI-powered analysis
	analyzer := dream.NewAnalyzer(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := analyzer.Analyze(ctx)
	if err != nil {
		return fmt.Sprintf("Error during analysis: %v\n\nPartial results:\n%s", err, dream.FormatResult(result))
	}

	return dream.FormatResult(result)
}

func (g *queueGUI) addDreamAsTask(dreamResult string) {
	// Parse dream result to extract feature name and description
	lines := strings.Split(dreamResult, "\n")
	var name, desc string

	for _, line := range lines {
		if strings.HasPrefix(line, "Feature:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Feature:"))
		} else if strings.HasPrefix(line, "Description:") {
			desc = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	if name == "" {
		name = "dream-feature"
	}
	if desc == "" {
		desc = dreamResult
	}

	task, err := g.queue.Add(name, desc)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	g.refreshTaskList()
	g.setStatus(fmt.Sprintf("Added dream task: %s", task.Name))
	dialog.ShowInformation("Task Added", fmt.Sprintf("Dream feature '%s' added to queue!", name), g.window)
}

func (g *queueGUI) setStatus(msg string) {
	g.statusBar.SetText(msg)
}

// Menu bar integration
func runMenuBar(q *queue.Queue, app fyne.App) {
	systray.Run(func() {
		systray.SetTitle("WS")
		systray.SetTooltip("WS Task Queue Manager")

		// Menu items
		mShow := systray.AddMenuItem("Show Queue", "Open queue window")
		systray.AddSeparator()

		mNextTask := systray.AddMenuItem("Process Next Task", "Start next pending task")
		mAddTask := systray.AddMenuItem("Add Task...", "Add new task to queue")
		systray.AddSeparator()

		mDream := systray.AddMenuItem("✨ Dream Feature", "AI suggests a feature")
		systray.AddSeparator()

		mQuit := systray.AddMenuItem("Quit", "Quit WS Queue")

		// Handle clicks
		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					// Show window
					for _, win := range app.Driver().AllWindows() {
						win.Show()
					}

				case <-mNextTask.ClickedCh:
					next := q.GetNextPending()
					if next != nil {
						now := time.Now()
						next.Status = queue.StatusRunning
						next.StartedAt = &now
						q.UpdateTask(next)

						go func() {
							cmd := exec.Command("ws", "ez", next.Name)
							cmd.Run()
						}()
					}

				case <-mAddTask.ClickedCh:
					// Show add task dialog
					for _, win := range app.Driver().AllWindows() {
						win.Show()
					}

				case <-mDream.ClickedCh:
					// Trigger dream analysis
					for _, win := range app.Driver().AllWindows() {
						win.Show()
					}

				case <-mQuit.ClickedCh:
					systray.Quit()
					app.Quit()
					return
				}
			}
		}()
	}, func() {})
}
