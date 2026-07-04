package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/tui/components/drawer"
)

type fakeRunner struct{ stdout, stderr string; err error }

func (r *fakeRunner) Run(_ context.Context, _ string) (string, string, error) {
	return r.stdout, r.stderr, r.err
}

func TestHandleCommandMode_EnterExecutesCommand(t *testing.T) {
	run := &fakeRunner{stdout: "ok"}
	m := &Model{
		mode:        modeCommand,
		searchQuery: "sync --full",
		runner:      run,
		tasks:       newTasksHolder(),
	}
	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 13})
	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	if _, ok := msg.(TaskFinishedMsg); !ok {
		t.Fatalf("expected TaskFinishedMsg, got %T", msg)
	}
	if m.mode != modeNormal {
		t.Fatal("mode should be normal after enter")
	}
	if m.searchQuery != "" {
		t.Fatal("searchQuery should be cleared")
	}
}

func TestHandleCommandMode_EscExitsCommandMode(t *testing.T) {
	m := &Model{
		mode:        modeCommand,
		searchQuery: "test",
	}
	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 27})
	if cmd != nil {
		t.Fatal("expected nil command")
	}
	if m.mode != modeNormal {
		t.Fatal("mode should be normal after esc")
	}
	if m.searchQuery != "" {
		t.Fatal("searchQuery should be cleared")
	}
}

func TestHandleCommandMode_QuitViaColonQ(t *testing.T) {
	m := &Model{
		mode:        modeCommand,
		searchQuery: "q",
		runner:      &fakeRunner{},
		tasks:       newTasksHolder(),
	}
	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 13})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestHandleCommandMode_BackspaceRemovesLastChar(t *testing.T) {
	m := &Model{
		mode:        modeCommand,
		searchQuery: "sync",
	}
	m.handleCommandMode(tea.KeyPressMsg{Code: 127})
	if m.searchQuery != "syn" {
		t.Fatalf("expected 'syn', got %q", m.searchQuery)
	}
}

func TestHandleCommandMode_EmptyCommandDoesNothing(t *testing.T) {
	m := &Model{
		mode:        modeCommand,
		searchQuery: "",
		runner:      &fakeRunner{},
		tasks:       newTasksHolder(),
	}
	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 13})
	if cmd != nil {
		t.Fatal("expected nil command for empty input")
	}
}

func TestExecuteCommandWithRunner(t *testing.T) {
	run := &fakeRunner{stdout: "hi"}
	m := &Model{
		tasks:  newTasksHolder(),
		runner: run,
		drawer: drawer.NewModel(),
	}
	cmd := m.executeCommand("test", "testing")
	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	tm, ok := msg.(TaskFinishedMsg)
	if !ok {
		t.Fatalf("expected TaskFinishedMsg, got %T", msg)
	}
	if tm.Message != "testing done" {
		t.Fatalf("expected 'testing done', got %q", tm.Message)
	}
}

func TestExecuteCommandWithRunner_Error(t *testing.T) {
	run := &fakeRunner{err: &testError{msg: "boom"}, stderr: "details"}
	m := &Model{
		tasks:  newTasksHolder(),
		runner: run,
		drawer: drawer.NewModel(),
	}
	cmd := m.executeCommand("test", "testing")
	msg := cmd()
	tm, ok := msg.(TaskFinishedMsg)
	if !ok {
		t.Fatalf("expected TaskFinishedMsg, got %T", msg)
	}
	if tm.Err == nil {
		t.Fatal("expected error")
	}
}
