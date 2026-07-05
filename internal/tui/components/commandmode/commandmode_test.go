package commandmode

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestCommandMode_EscExits(t *testing.T) {
	m := NewModel()
	m.focused = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: escapeKey})
	sm := updated.(Model)
	if sm.focused {
		t.Error("expected focused=false after Esc")
	}
	if cmd != nil {
		t.Error("expected nil cmd from Esc")
	}
}

func TestCommandMode_EnterExecutesCommand(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "sync --full"
	m.cursorPos = len(m.input)
	updated, cmd := m.Update(tea.KeyPressMsg{Code: enterKey})
	sm := updated.(Model)
	if sm.focused {
		t.Error("expected focused=false after Enter")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	msg := cmd()
	execMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatalf("expected CommandExecutedMsg, got %T", msg)
	}
	if execMsg.Input != "sync --full" {
		t.Errorf("expected input 'sync --full', got '%s'", execMsg.Input)
	}
}

func TestCommandMode_BackspaceDeletesAtCursor(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "sync"
	m.cursorPos = len(m.input)
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.input != "syn" {
		t.Errorf("expected 'syn' after backspace, got '%s'", sm.input)
	}
	if sm.cursorPos != 3 {
		t.Errorf("expected cursorPos=3 after backspace, got %d", sm.cursorPos)
	}
}

func TestCommandMode_BackspaceOnEmpty(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = ""
	m.cursorPos = 0
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.input != "" {
		t.Errorf("expected empty input, got '%s'", sm.input)
	}
}

func TestCommandMode_TypingAppendsRunes(t *testing.T) {
	m := NewModel()
	m.focused = true
	updated, _ := m.Update(tea.KeyPressMsg{Code: 's'})
	sm := updated.(Model)
	if sm.input != "s" {
		t.Errorf("expected 's', got '%s'", sm.input)
	}
	if sm.cursorPos != 1 {
		t.Errorf("expected cursorPos=1, got %d", sm.cursorPos)
	}
	updated2, _ := sm.Update(tea.KeyPressMsg{Code: 'y'})
	sm2 := updated2.(Model)
	if sm2.input != "sy" {
		t.Errorf("expected 'sy', got '%s'", sm2.input)
	}
	if sm2.cursorPos != 2 {
		t.Errorf("expected cursorPos=2, got %d", sm2.cursorPos)
	}
}

func TestCommandMode_UnfocusedIgnoresKeystrokes(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyPressMsg{Code: 's'})
	sm := updated.(Model)
	if sm.input != "" {
		t.Errorf("expected empty input when unfocused, got '%s'", sm.input)
	}
}

func TestCommandMode_View_Unfocused(t *testing.T) {
	m := NewModel()
	got := m.View().Content
	if got != "" {
		t.Errorf("expected empty view when unfocused, got '%s'", got)
	}
}

func TestCommandMode_View_Focused(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "sync"
	m.cursorPos = len(m.input)
	got := stripANSI(m.View().Content)
	if !strings.Contains(got, "Execute Command") {
		t.Errorf("expected title 'Execute Command' in view, got: %s", got)
	}
}

func TestCommandMode_IsFocused(t *testing.T) {
	m := NewModel()
	if m.IsFocused() {
		t.Error("expected IsFocused=false initially")
	}
	m.focused = true
	if !m.IsFocused() {
		t.Error("expected IsFocused=true after setting focused")
	}
}

func TestCommandMode_LeftRightMovesCursor(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "abc"
	m.cursorPos = len(m.input)

	m1, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m1.(Model).cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after Left, got %d", m1.(Model).cursorPos)
	}

	m2, _ := m1.(Model).Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m2.(Model).cursorPos != 1 {
		t.Errorf("expected cursorPos=1 after Left, got %d", m2.(Model).cursorPos)
	}

	m3, _ := m2.(Model).Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m3.(Model).cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after Right, got %d", m3.(Model).cursorPos)
	}
}

func TestCommandMode_LeftRightClampsBounds(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "a"
	m.cursorPos = 0

	m1, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m1.(Model).cursorPos != 0 {
		t.Errorf("expected cursorPos=0 at left bound, got %d", m1.(Model).cursorPos)
	}

	m.cursorPos = 1
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m2.(Model).cursorPos != 1 {
		t.Errorf("expected cursorPos=1 at right bound, got %d", m2.(Model).cursorPos)
	}
}

func TestCommandMode_HomeEndMovesCursor(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "hello world"
	m.cursorPos = 3

	m1, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if m1.(Model).cursorPos != 0 {
		t.Errorf("expected cursorPos=0 after Home, got %d", m1.(Model).cursorPos)
	}

	m2, _ := m1.(Model).Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if m2.(Model).cursorPos != 11 {
		t.Errorf("expected cursorPos=11 after End, got %d", m2.(Model).cursorPos)
	}
}

func TestCommandMode_InsertAtCursorPosition(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "ab"
	m.cursorPos = 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'X'})
	sm := updated.(Model)
	if sm.input != "aXb" {
		t.Errorf("expected 'aXb' after insert at pos 1, got '%s'", sm.input)
	}
	if sm.cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after insert, got %d", sm.cursorPos)
	}
}

func TestCommandMode_DeleteAtCursorPosition(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "abcd"
	m.cursorPos = 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	sm := updated.(Model)
	if sm.input != "acd" {
		t.Errorf("expected 'acd' after delete at pos 1, got '%s'", sm.input)
	}
	if sm.cursorPos != 1 {
		t.Errorf("expected cursorPos=1 after delete, got %d", sm.cursorPos)
	}

	// Delete at end should be a no-op
	sm.cursorPos = 3
	updated2, _ := sm.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	sm2 := updated2.(Model)
	if sm2.input != "acd" {
		t.Errorf("expected 'acd' unchanged after delete at end, got '%s'", sm2.input)
	}
}
