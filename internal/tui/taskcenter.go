package tui

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

type TaskState string

const (
	TaskQueued   TaskState = "queued"
	TaskRunning  TaskState = "running"
	TaskSuccess  TaskState = "success"
	TaskFailed   TaskState = "failed"
	TaskCanceled TaskState = "cancelled"
)

type Task struct {
	ID    string
	Label string
	State TaskState
	Err   error
}

type TaskCenter struct {
	mu      sync.RWMutex
	tasks   map[string]*Task
	counter uint64
}

func NewTaskCenter() *TaskCenter {
	return &TaskCenter{
		tasks: make(map[string]*Task),
	}
}

func (tc *TaskCenter) Enqueue(label string) string {
	id := fmt.Sprintf("task-%d", atomic.AddUint64(&tc.counter, 1))
	tc.mu.Lock()
	tc.tasks[id] = &Task{ID: id, Label: label, State: TaskQueued}
	tc.mu.Unlock()
	return id
}

func (tc *TaskCenter) MarkRunning(id string) {
	tc.mu.Lock()
	if t, ok := tc.tasks[id]; ok {
		t.State = TaskRunning
	}
	tc.mu.Unlock()
}

func (tc *TaskCenter) MarkDone(id string, err error) {
	tc.mu.Lock()
	if t, ok := tc.tasks[id]; ok {
		if err != nil {
			t.State = TaskFailed
			t.Err = err
		} else {
			t.State = TaskSuccess
		}
	}
	tc.mu.Unlock()
}

func (tc *TaskCenter) Cancel(id string) {
	tc.mu.Lock()
	if t, ok := tc.tasks[id]; ok {
		t.State = TaskCanceled
	}
	tc.mu.Unlock()
}

func (tc *TaskCenter) State(id string) TaskState {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if t, ok := tc.tasks[id]; ok {
		return t.State
	}
	return ""
}

func (tc *TaskCenter) Summary() string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	var parts []string
	var queued, running, success, failed, cancelled int
	for _, t := range tc.tasks {
		switch t.State {
		case TaskQueued:
			queued++
		case TaskRunning:
			running++
		case TaskSuccess:
			success++
		case TaskFailed:
			failed++
		case TaskCanceled:
			cancelled++
		}
	}
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	if success > 0 {
		parts = append(parts, fmt.Sprintf("%d done", success))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if cancelled > 0 {
		parts = append(parts, fmt.Sprintf("%d cancelled", cancelled))
	}
	if queued > 0 {
		parts = append(parts, fmt.Sprintf("%d queued", queued))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}
