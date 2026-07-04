package commandinput

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func goldenPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", t.Name()+".golden")
}

func updateGolden() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

func TestCommandInput_EscExitsCommandMode(t *testing.T) {
	m := NewModel()
	m.focused = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: escapeKey})
	sm := updated.(Model)
	if sm.focused {
		t.Error("expected command mode to exit after Esc")
	}
	if cmd != nil {
		t.Error("expected nil cmd from Esc")
	}
}

func TestCommandInput_EnterExecutesCommand(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "sync --full"
	updated, cmd := m.Update(tea.KeyPressMsg{Code: enterKey})
	sm := updated.(Model)
	if sm.focused {
		t.Error("expected command mode to exit after Enter")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	msg := cmd()
	execMsg, ok := msg.(CommandExecutedMsg)
	if !ok {
		t.Fatalf("expected CommandExecutedMsg, got %T", msg)
	}
	if execMsg.Command != "sync --full" {
		t.Errorf("expected command 'sync --full', got '%s'", execMsg.Command)
	}
}

func TestCommandInput_EnterEmptyDoesNotAddHistory(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = ""
	updated, cmd := m.Update(tea.KeyPressMsg{Code: enterKey})
	sm := updated.(Model)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter (even with empty command)")
	}
	if len(sm.history) != 0 {
		t.Errorf("expected empty history for empty command, got %v", sm.history)
	}
}

func TestCommandInput_HistoryUpDownNavigation(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "sync"
	updated, _ := m.Update(tea.KeyPressMsg{Code: enterKey})
	m = updated.(Model)
	m.focused = true
	m.query = "analyze"
	updated, _ = m.Update(tea.KeyPressMsg{Code: enterKey})
	m = updated.(Model)
	m.focused = true
	m.query = ""

	// Up → should show "analyze" (latest)
	updated, _ = m.Update(tea.KeyPressMsg{Code: upKey})
	m = updated.(Model)
	if m.query != "analyze" {
		t.Errorf("expected 'analyze' after first up, got '%s'", m.query)
	}

	// Up again → should show "sync"
	updated, _ = m.Update(tea.KeyPressMsg{Code: upKey})
	m = updated.(Model)
	if m.query != "sync" {
		t.Errorf("expected 'sync' after second up, got '%s'", m.query)
	}

	// Up again at boundary → should stay at "sync" (position 0)
	updated, _ = m.Update(tea.KeyPressMsg{Code: upKey})
	m = updated.(Model)
	if m.query != "sync" {
		t.Errorf("expected 'sync' at boundary, got '%s'", m.query)
	}

	// Down → back to "analyze"
	updated, _ = m.Update(tea.KeyPressMsg{Code: downKey})
	m = updated.(Model)
	if m.query != "analyze" {
		t.Errorf("expected 'analyze' after down, got '%s'", m.query)
	}

	// Down again → go to new editing (position -1)
	updated, _ = m.Update(tea.KeyPressMsg{Code: downKey})
	m = updated.(Model)
	if m.query != "" {
		t.Errorf("expected empty query after going past end, got '%s'", m.query)
	}

	// Down again at -1 → no-op
	updated, _ = m.Update(tea.KeyPressMsg{Code: downKey})
	m = updated.(Model)
	if m.query != "" {
		t.Errorf("expected empty query at no-op, got '%s'", m.query)
	}
}

func TestCommandInput_HistoryNoHistory(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.Update(tea.KeyPressMsg{Code: upKey})
	if m.query != "" {
		t.Errorf("expected empty query when no history, got '%s'", m.query)
	}
	m.Update(tea.KeyPressMsg{Code: downKey})
	if m.query != "" {
		t.Errorf("expected empty query when no history, got '%s'", m.query)
	}
}

func TestCommandInput_BackspaceDeletesChar(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "go"
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.query != "g" {
		t.Errorf("expected 'g' after backspace, got '%s'", sm.query)
	}
}

func TestCommandInput_BackspaceOnEmpty(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = ""
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.query != "" {
		t.Errorf("expected empty query, got '%s'", sm.query)
	}
}

func TestCommandInput_TypingAppendsRunes(t *testing.T) {
	m := NewModel()
	m.focused = true
	updated, _ := m.Update(tea.KeyPressMsg{Code: 's'})
	sm := updated.(Model)
	if sm.query != "s" {
		t.Errorf("expected 's', got '%s'", sm.query)
	}
	updated2, _ := sm.Update(tea.KeyPressMsg{Code: 'y'})
	sm2 := updated2.(Model)
	if sm2.query != "sy" {
		t.Errorf("expected 'sy', got '%s'", sm2.query)
	}
}

func TestCommandInput_UnfocusedIgnoresKeystrokes(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'g'})
	sm := updated.(Model)
	if sm.query != "" {
		t.Errorf("expected empty query when unfocused, got '%s'", sm.query)
	}
}

func TestCommandInput_View_Unfocused(t *testing.T) {
	m := NewModel()
	got := m.View().Content
	if got != "" {
		t.Errorf("expected empty view when unfocused, got '%s'", got)
	}
}

func TestCommandInput_View_Focused(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "sync"
	got := stripANSI(m.View().Content)
	if got == "" {
		t.Error("expected non-empty view when focused")
	}
}

func TestCommandInput_View_Golden(t *testing.T) {
	tests := []struct {
		name    string
		focused bool
		query   string
	}{
		{"focused_empty", true, ""},
		{"focused_query", true, "sync --full"},
		{"unfocused", false, "sync"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.focused = tt.focused
			m.query = tt.query
			got := stripANSI(m.View().Content)

			path := goldenPath(t)
			if updateGolden() {
				err := os.MkdirAll(filepath.Dir(path), 0o755)
				if err != nil {
					t.Fatalf("failed to create testdata dir: %v", err)
				}
				err = os.WriteFile(path, []byte(got), 0o644)
				if err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read golden file: %v (run with UPDATE_GOLDEN=1 to generate)", err)
			}
			if string(want) != got {
				t.Errorf("golden mismatch:\n--- want:\n%s\n--- got:\n%s", string(want), got)
			}
		})
	}
}

func TestCommandInput_IsFocused(t *testing.T) {
	m := NewModel()
	if m.IsFocused() {
		t.Error("expected IsFocused=false initially")
	}
	m.focused = true
	if !m.IsFocused() {
		t.Error("expected IsFocused=true after setting focused")
	}
}

func TestCommandInput_SetFocusedResetsState(t *testing.T) {
	m := NewModel()
	// Simulate having entered a command and browsing history
	m.history = []string{"sync", "analyze"}
	m.historyPos = 0
	m.query = "sync"
	m.SetFocused(true)
	if m.historyPos != -1 {
		t.Errorf("expected historyPos=-1 after SetFocused(true), got %d", m.historyPos)
	}
	if m.query != "" {
		t.Errorf("expected empty query after SetFocused(true), got '%s'", m.query)
	}
}

func TestCommandInput_EscResetsHistoryPos(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "sync"
	updated, _ := m.Update(tea.KeyPressMsg{Code: enterKey})
	m = updated.(Model)
	m.focused = true
	m.query = "analyze"
	updated, _ = m.Update(tea.KeyPressMsg{Code: enterKey})
	m = updated.(Model)
	m.focused = true
	// Navigate to history
	updated, _ = m.Update(tea.KeyPressMsg{Code: upKey})
	m = updated.(Model)
	if m.historyPos == -1 {
		t.Fatal("expected historyPos to not be -1 after up")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: escapeKey})
	m = updated.(Model)
	if m.historyPos != -1 {
		t.Errorf("expected historyPos=-1 after Esc, got %d", m.historyPos)
	}
	if m.focused {
		t.Error("expected not focused after Esc")
	}
}
