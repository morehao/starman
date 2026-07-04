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

func TestCommandMode_BackspaceDeletesChar(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = "sync"
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.input != "syn" {
		t.Errorf("expected 'syn' after backspace, got '%s'", sm.input)
	}
}

func TestCommandMode_BackspaceOnEmpty(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.input = ""
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
	updated2, _ := sm.Update(tea.KeyPressMsg{Code: 'y'})
	sm2 := updated2.(Model)
	if sm2.input != "sy" {
		t.Errorf("expected 'sy', got '%s'", sm2.input)
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
