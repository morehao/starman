package tui

import (
	"errors"
	"strings"
	"testing"
)

func TestTaskCenterLifecycle(t *testing.T) {
	tc := NewTaskCenter()
	id := tc.Enqueue("sync")

	if tc.State(id) != TaskQueued {
		t.Fatalf("expected TaskQueued, got %s", tc.State(id))
	}

	tc.MarkRunning(id)
	if tc.State(id) != TaskRunning {
		t.Fatalf("expected TaskRunning, got %s", tc.State(id))
	}

	tc.MarkDone(id, nil)
	if tc.State(id) != TaskSuccess {
		t.Fatalf("expected TaskSuccess, got %s", tc.State(id))
	}
}

func TestTaskCenterFailedTask(t *testing.T) {
	tc := NewTaskCenter()
	id := tc.Enqueue("analyze")

	tc.MarkRunning(id)
	tc.MarkDone(id, errors.New("something broke"))

	if tc.State(id) != TaskFailed {
		t.Fatalf("expected TaskFailed, got %s", tc.State(id))
	}
}

func TestTaskCenterSummary(t *testing.T) {
	tc := NewTaskCenter()

	id1 := tc.Enqueue("sync")
	tc.MarkRunning(id1)

	id2 := tc.Enqueue("analyze")
	tc.MarkRunning(id2)
	tc.MarkDone(id2, nil)

	id3 := tc.Enqueue("generate")
	tc.MarkRunning(id3)
	tc.MarkDone(id3, errors.New("fail"))

	summary := tc.Summary()

	if !strings.Contains(summary, "1 running") {
		t.Fatalf("expected '1 running' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1 done") {
		t.Fatalf("expected '1 done' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1 failed") {
		t.Fatalf("expected '1 failed' in summary, got %q", summary)
	}
}

func TestTaskCenterCancellation(t *testing.T) {
	tc := NewTaskCenter()
	id := tc.Enqueue("backup")

	if tc.State(id) != TaskQueued {
		t.Fatalf("expected TaskQueued, got %s", tc.State(id))
	}

	tc.Cancel(id)
	if tc.State(id) != TaskCanceled {
		t.Fatalf("expected TaskCanceled, got %s", tc.State(id))
	}
}

func TestTaskCenterEmptySummary(t *testing.T) {
	tc := NewTaskCenter()

	if s := tc.Summary(); s != "" {
		t.Fatalf("expected empty summary, got %q", s)
	}
}

func TestTaskCenterUnknownState(t *testing.T) {
	tc := NewTaskCenter()

	if s := tc.State("nonexistent"); s != "" {
		t.Fatalf("expected empty state for unknown id, got %q", s)
	}
}

func TestTaskCenterMarkDonePreservesErr(t *testing.T) {
	tc := NewTaskCenter()
	id := tc.Enqueue("sync")
	tc.MarkRunning(id)
	testErr := errors.New("test error")
	tc.MarkDone(id, testErr)

	tc.mu.RLock()
	task := tc.tasks[id]
	tc.mu.RUnlock()

	if task.Err != testErr {
		t.Fatalf("expected err to be preserved, got %v", task.Err)
	}
}
