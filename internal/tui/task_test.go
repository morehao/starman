package tui

import (
	"testing"
)

func TestTasksHolderStartAndFinish(t *testing.T) {
	tasks := newTasksHolder()
	if !tasks.isEmpty() {
		t.Fatal("expected empty")
	}

	tasks.start("sync-1", "sync")
	if tasks.isEmpty() {
		t.Fatal("expected non-empty after start")
	}

	task := tasks.latest()
	if task == nil {
		t.Fatal("expected task")
	}
	if task.Status != TaskRunning {
		t.Fatalf("expected running, got %d", task.Status)
	}
	if task.Name != "sync" {
		t.Fatalf("expected sync, got %s", task.Name)
	}
}

func TestTasksHolderFinishSuccess(t *testing.T) {
	tasks := newTasksHolder()
	tasks.start("sync-1", "sync")
	tasks.finish("sync-1", "synced 10 repos", nil)

	task := tasks.latest()
	if task.Status != TaskSuccess {
		t.Fatalf("expected success, got %d", task.Status)
	}
	if task.Message != "synced 10 repos" {
		t.Fatalf("unexpected message: %s", task.Message)
	}
	if task.Err != nil {
		t.Fatal("expected nil error")
	}
}

func TestTasksHolderFinishError(t *testing.T) {
	tasks := newTasksHolder()
	tasks.start("sync-1", "sync")
	tasks.finish("sync-1", "sync failed", errTest)

	task := tasks.latest()
	if task.Status != TaskFailed {
		t.Fatalf("expected failed, got %d", task.Status)
	}
	if task.Err != errTest {
		t.Fatal("expected error")
	}
}

func TestTasksHolderClear(t *testing.T) {
	tasks := newTasksHolder()
	tasks.start("task-1", "test")
	tasks.clear("task-1")
	if !tasks.isEmpty() {
		t.Fatal("expected empty after clear")
	}
}

func TestTasksHolderMultipleTasks(t *testing.T) {
	tasks := newTasksHolder()
	tasks.start("task-1", "first")
	tasks.start("task-2", "second")

	latest := tasks.latest()
	if latest.Name != "second" {
		t.Fatalf("expected second, got %s", latest.Name)
	}
}

var errTest = &testError{msg: "test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
