package queue

import (
	"time"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	StatusQueued     TaskStatus = "queued"
	StatusRunning    TaskStatus = "running"
	StatusValidating TaskStatus = "validating"
	StatusConflict   TaskStatus = "conflict"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

// Task represents a single development task in the queue
type Task struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// IsActive returns true if the task is currently being processed
func (t *Task) IsActive() bool {
	return t.Status == StatusRunning || t.Status == StatusValidating || t.Status == StatusConflict
}

// IsPending returns true if the task is waiting to be processed
func (t *Task) IsPending() bool {
	return t.Status == StatusQueued
}

// IsComplete returns true if the task has finished (success or failure)
func (t *Task) IsComplete() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed
}
