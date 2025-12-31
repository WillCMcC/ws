package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Queue manages a list of development tasks
type Queue struct {
	Tasks    []Task `json:"tasks"`
	filePath string
}

// NewQueue creates a new queue instance
func NewQueue() (*Queue, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	wsDir := filepath.Join(configDir, "ws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create ws config directory: %w", err)
	}

	filePath := filepath.Join(wsDir, "queue.json")
	q := &Queue{
		Tasks:    []Task{},
		filePath: filePath,
	}

	// Load existing queue if it exists
	if _, err := os.Stat(filePath); err == nil {
		if err := q.Load(); err != nil {
			return nil, err
		}
	}

	return q, nil
}

// Load reads the queue from disk
func (q *Queue) Load() error {
	data, err := os.ReadFile(q.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read queue file: %w", err)
	}

	if err := json.Unmarshal(data, &q); err != nil {
		return fmt.Errorf("failed to parse queue file: %w", err)
	}

	return nil
}

// Save writes the queue to disk
func (q *Queue) Save() error {
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queue: %w", err)
	}

	if err := os.WriteFile(q.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write queue file: %w", err)
	}

	return nil
}

// Add adds a new task to the queue
func (q *Queue) Add(name, description string) (*Task, error) {
	task := Task{
		ID:          uuid.New().String()[:8],
		Name:        name,
		Description: description,
		Status:      StatusQueued,
		CreatedAt:   time.Now(),
	}

	q.Tasks = append(q.Tasks, task)
	if err := q.Save(); err != nil {
		return nil, err
	}

	return &task, nil
}

// GetByID returns a task by its ID
func (q *Queue) GetByID(id string) *Task {
	for i := range q.Tasks {
		if q.Tasks[i].ID == id {
			return &q.Tasks[i]
		}
	}
	return nil
}

// GetByName returns a task by its name
func (q *Queue) GetByName(name string) *Task {
	for i := range q.Tasks {
		if q.Tasks[i].Name == name {
			return &q.Tasks[i]
		}
	}
	return nil
}

// UpdateTask updates a task in the queue
func (q *Queue) UpdateTask(task *Task) error {
	for i := range q.Tasks {
		if q.Tasks[i].ID == task.ID {
			q.Tasks[i] = *task
			return q.Save()
		}
	}
	return fmt.Errorf("task not found: %s", task.ID)
}

// Remove removes a task from the queue
func (q *Queue) Remove(id string) error {
	for i := range q.Tasks {
		if q.Tasks[i].ID == id {
			q.Tasks = append(q.Tasks[:i], q.Tasks[i+1:]...)
			return q.Save()
		}
	}
	return fmt.Errorf("task not found: %s", id)
}

// GetNextPending returns the next pending task
func (q *Queue) GetNextPending() *Task {
	for i := range q.Tasks {
		if q.Tasks[i].IsPending() {
			return &q.Tasks[i]
		}
	}
	return nil
}

// GetActive returns the currently active task
func (q *Queue) GetActive() *Task {
	for i := range q.Tasks {
		if q.Tasks[i].IsActive() {
			return &q.Tasks[i]
		}
	}
	return nil
}

// Clear removes all completed and failed tasks
func (q *Queue) Clear() error {
	newTasks := []Task{}
	for _, task := range q.Tasks {
		if !task.IsComplete() {
			newTasks = append(newTasks, task)
		}
	}
	q.Tasks = newTasks
	return q.Save()
}
