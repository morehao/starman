package searchinput

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

func TestSearchInput_EscExitsSearchMode(t *testing.T) {
	m := NewModel()
	m.focused = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: escapeKey})
	sm := updated.(Model)
	if sm.focused {
		t.Error("expected search mode to exit after Esc")
	}
	if cmd != nil {
		t.Error("expected nil cmd from Esc")
	}
}

func TestSearchInput_EnterExecutesSearch(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "go cli"
	updated, cmd := m.Update(tea.KeyPressMsg{Code: enterKey})
	sm := updated.(Model)
	if sm.focused {
		t.Error("expected search mode to exit after Enter")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	msg := cmd()
	searchMsg, ok := msg.(SearchExecutedMsg)
	if !ok {
		t.Fatalf("expected SearchExecutedMsg, got %T", msg)
	}
	if searchMsg.Query != "go cli" {
		t.Errorf("expected query 'go cli', got '%s'", searchMsg.Query)
	}
}

func TestSearchInput_BackspaceDeletesChar(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "go"
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.query != "g" {
		t.Errorf("expected 'g' after backspace, got '%s'", sm.query)
	}
}

func TestSearchInput_BackspaceOnEmpty(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = ""
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	sm := updated.(Model)
	if sm.query != "" {
		t.Errorf("expected empty query, got '%s'", sm.query)
	}
}

func TestSearchInput_TypingAppendsRunes(t *testing.T) {
	m := NewModel()
	m.focused = true
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'g'})
	sm := updated.(Model)
	if sm.query != "g" {
		t.Errorf("expected 'g', got '%s'", sm.query)
	}
	updated2, _ := sm.Update(tea.KeyPressMsg{Code: 'o'})
	sm2 := updated2.(Model)
	if sm2.query != "go" {
		t.Errorf("expected 'go', got '%s'", sm2.query)
	}
}

func TestSearchInput_UnfocusedIgnoresKeystrokes(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'g'})
	sm := updated.(Model)
	if sm.query != "" {
		t.Errorf("expected empty query when unfocused, got '%s'", sm.query)
	}
}

func TestSearchInput_View_Unfocused(t *testing.T) {
	m := NewModel()
	got := m.View().Content
	if got != "" {
		t.Errorf("expected empty view when unfocused, got '%s'", got)
	}
}

func TestSearchInput_View_Focused(t *testing.T) {
	m := NewModel()
	m.focused = true
	m.query = "go"
	got := stripANSI(m.View().Content)
	if got == "" {
		t.Error("expected non-empty view when focused")
	}
}

func TestSearchInput_View_Golden(t *testing.T) {
	tests := []struct {
		name    string
		focused bool
		query   string
	}{
		{"focused_empty", true, ""},
		{"focused_query", true, "go cli framework"},
		{"unfocused", false, "go"},
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

func TestSearchInput_IsFocused(t *testing.T) {
	m := NewModel()
	if m.IsFocused() {
		t.Error("expected IsFocused=false initially")
	}
	m.focused = true
	if !m.IsFocused() {
		t.Error("expected IsFocused=true after setting focused")
	}
}
