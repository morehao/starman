package tui

import (
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type TaskStatus int

const (
	TaskRunning TaskStatus = iota
	TaskSuccess
	TaskFailed
)

type Task struct {
	ID        string
	Name      string
	Status    TaskStatus
	Message   string
	Err       error
	StartTime time.Time
	EndTime   time.Time
}

type TaskStartedMsg struct {
	TaskID string
	Name   string
}

type TaskProgressMsg struct {
	TaskID  string
	Message string
}

type TaskFinishedMsg struct {
	TaskID  string
	Name    string
	Message string
	Err     error
}

type TaskClearedMsg struct {
	TaskID string
}

type ErrorClearedMsg struct{}

type tasksHolder struct {
	mu    sync.Mutex
	items map[string]*Task
	order []string
}

func newTasksHolder() *tasksHolder {
	return &tasksHolder{items: make(map[string]*Task)}
}

func (t *tasksHolder) start(id, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.items[id] = &Task{
		ID:        id,
		Name:      name,
		Status:    TaskRunning,
		StartTime: time.Now(),
	}
	t.order = append(t.order, id)
}

func (t *tasksHolder) finish(id string, msg string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if task, ok := t.items[id]; ok {
		task.EndTime = time.Now()
		task.Message = msg
		task.Err = err
		if err != nil {
			task.Status = TaskFailed
		} else {
			task.Status = TaskSuccess
		}
	}
}

func (t *tasksHolder) clear(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.items, id)
	for i, oid := range t.order {
		if oid == id {
			t.order = append(t.order[:i], t.order[i+1:]...)
			break
		}
	}
}

func (t *tasksHolder) latest() *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.order) == 0 {
		return nil
	}
	return t.items[t.order[len(t.order)-1]]
}

func (t *tasksHolder) isEmpty() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.items) == 0
}

type spinnerTickMsg struct {
	frame int
}

func tickSpinner() tea.Cmd {
	frame := 0
	return tea.Tick(time.Millisecond*120, func(t time.Time) tea.Msg {
		frame = (frame + 1) % len(spinnerFrames)
		return spinnerTickMsg{frame: frame}
	})
}
