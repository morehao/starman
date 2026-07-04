package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/tui/components/commandmode"
	"github.com/morehao/starman/internal/tui/components/drawer"
)

type fakeRunner struct{ stdout, stderr string; err error }

func (r *fakeRunner) Run(_ context.Context, _ string) (string, string, error) {
	return r.stdout, r.stderr, r.err
}

func TestCommandMode_EnterExecutesCommand(t *testing.T) {
	run := &fakeRunner{stdout: "ok"}
	m := &Model{
		mode:         modeCommand,
		commandInput: commandmode.NewModel(),
		runner:       run,
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)
	m.commandInput.SetSize(100, 40)
	m.commandInput.Update(tea.KeyPressMsg{Code: 's'})
	m.commandInput.Update(tea.KeyPressMsg{Code: 'y'})
	m.commandInput.Update(tea.KeyPressMsg{Code: 'n'})
	m.commandInput.Update(tea.KeyPressMsg{Code: 'c'})

	updated, cmd := m.commandInput.Update(tea.KeyPressMsg{Code: 13})
	m.commandInput = updated.(commandmode.Model)

	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	if _, ok := msg.(commandmode.CommandExecutedMsg); !ok {
		t.Fatalf("expected CommandExecutedMsg, got %T", msg)
	}
	if m.commandInput.IsFocused() {
		t.Fatal("command input should not be focused after Enter")
	}
}

func TestCommandMode_EscExitsCommandMode(t *testing.T) {
	m := &Model{
		mode:         modeCommand,
		commandInput: commandmode.NewModel(),
	}
	m.commandInput.SetFocused(true)

	updated, cmd := m.commandInput.Update(tea.KeyPressMsg{Code: 27})
	m.commandInput = updated.(commandmode.Model)

	if cmd != nil {
		t.Fatal("expected nil command")
	}
	if m.commandInput.IsFocused() {
		t.Fatal("command input should not be focused after Esc")
	}
}

func TestCommandMode_EmptyInputDoesNothing(t *testing.T) {
	run := &fakeRunner{stdout: "ok"}
	m := &Model{
		mode:         modeCommand,
		commandInput: commandmode.NewModel(),
		runner:       run,
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)

	updated, cmd := m.commandInput.Update(tea.KeyPressMsg{Code: 13})
	m.commandInput = updated.(commandmode.Model)

	if cmd == nil {
		t.Fatal("expected a cmd even for empty (routing to updateInner handles the empty check)")
	}
	msg := cmd()
	execMsg, ok := msg.(commandmode.CommandExecutedMsg)
	if !ok {
		t.Fatalf("expected CommandExecutedMsg, got %T", msg)
	}
	if execMsg.Input != "" {
		t.Fatalf("expected empty input, got '%s'", execMsg.Input)
	}
}

func TestCommandMode_QuitViaCommandInput(t *testing.T) {
	m := &Model{
		mode:         modeCommand,
		commandInput: commandmode.NewModel(),
		runner:       &fakeRunner{},
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)
	updated, _ := m.commandInput.Update(tea.KeyPressMsg{Code: 'q'})
	m.commandInput = updated.(commandmode.Model)
	updated, cmd := m.commandInput.Update(tea.KeyPressMsg{Code: 13})
	m.commandInput = updated.(commandmode.Model)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	msg := cmd()
	execMsg, ok := msg.(commandmode.CommandExecutedMsg)
	if !ok {
		t.Fatalf("expected CommandExecutedMsg, got %T", msg)
	}
	if execMsg.Input != "q" {
		t.Fatalf("expected input 'q', got '%s'", execMsg.Input)
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
	if tm.Name != "testing" {
		t.Fatalf("expected Name 'testing', got %q", tm.Name)
	}
	if tm.Message != "hi" {
		t.Fatalf("expected Message 'hi', got %q", tm.Message)
	}
	if tm.Err != nil {
		t.Fatal("expected no error")
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
	if tm.Name != "testing" {
		t.Fatalf("expected Name 'testing', got %q", tm.Name)
	}
	if tm.Err == nil {
		t.Fatal("expected error")
	}
	if tm.Message != "details" {
		t.Fatalf("expected Message 'details', got %q", tm.Message)
	}
}
