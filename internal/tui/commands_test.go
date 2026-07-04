package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/tui/components/commandinput"
	"github.com/morehao/starman/internal/tui/components/drawer"
)

type fakeRunner struct{ stdout, stderr string; err error }

func (r *fakeRunner) Run(_ context.Context, _ string) (string, string, error) {
	return r.stdout, r.stderr, r.err
}

func TestHandleCommandMode_EnterExecutesCommand(t *testing.T) {
	run := &fakeRunner{stdout: "ok"}
	m := &Model{
		mode:         modeCommand,
		commandInput: commandinput.NewModel(),
		runner:       run,
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)

	m.handleCommandMode(tea.KeyPressMsg{Code: 's'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'y'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'n'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'c'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 32})
	m.handleCommandMode(tea.KeyPressMsg{Code: '-'})
	m.handleCommandMode(tea.KeyPressMsg{Code: '-'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'f'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'u'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'l'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'l'})

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
}

func TestHandleCommandMode_EscExitsCommandMode(t *testing.T) {
	m := &Model{
		mode:         modeCommand,
		commandInput: commandinput.NewModel(),
	}
	m.commandInput.SetFocused(true)

	m.handleCommandMode(tea.KeyPressMsg{Code: 't'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'e'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 's'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 't'})

	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 27})
	if cmd != nil {
		t.Fatal("expected nil command")
	}
	if m.mode != modeNormal {
		t.Fatal("mode should be normal after esc")
	}
	if m.commandInput.IsFocused() {
		t.Fatal("command input should not be focused after esc")
	}
}

func TestHandleCommandMode_QuitViaColonQ(t *testing.T) {
	m := &Model{
		mode:         modeCommand,
		commandInput: commandinput.NewModel(),
		runner:       &fakeRunner{},
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)

	m.handleCommandMode(tea.KeyPressMsg{Code: 'q'})

	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 13})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestHandleCommandMode_QuitViaColonQuit(t *testing.T) {
	m := &Model{
		mode:         modeCommand,
		commandInput: commandinput.NewModel(),
		runner:       &fakeRunner{},
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)

	m.handleCommandMode(tea.KeyPressMsg{Code: 'q'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'u'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 'i'})
	m.handleCommandMode(tea.KeyPressMsg{Code: 't'})

	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 13})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestHandleCommandMode_EmptyCommandDoesNothing(t *testing.T) {
	m := &Model{
		mode:         modeCommand,
		commandInput: commandinput.NewModel(),
		runner:       &fakeRunner{},
		tasks:        newTasksHolder(),
	}
	m.commandInput.SetFocused(true)
	cmd := m.handleCommandMode(tea.KeyPressMsg{Code: 13})
	if cmd != nil {
		t.Fatal("expected nil command for empty input")
	}
	if m.mode != modeNormal {
		t.Fatal("expected mode to be normal after empty command")
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
