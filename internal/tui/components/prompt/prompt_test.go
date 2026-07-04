package prompt

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

func TestPrompt_ConfirmYes(t *testing.T) {
	m := NewConfirmModel("Run full sync?")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: yKey})
	pm := updated.(Model)
	if pm.active {
		t.Error("expected prompt to deactivate after confirm")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from confirm")
	}
	msg := cmd().(PromptResultMsg)
	if !msg.Confirmed {
		t.Error("expected confirmed=true for yes")
	}
	if msg.Type != PromptConfirm {
		t.Errorf("expected PromptConfirm type, got %v", msg.Type)
	}
}

func TestPrompt_ConfirmYes_Uppercase(t *testing.T) {
	m := NewConfirmModel("Run full sync?")
	_, cmd := m.Update(tea.KeyPressMsg{Code: YKey})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Y key")
	}
	msg := cmd().(PromptResultMsg)
	if !msg.Confirmed {
		t.Error("expected confirmed=true for Y")
	}
}

func TestPrompt_ConfirmNo(t *testing.T) {
	m := NewConfirmModel("Run full sync?")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: nKey})
	pm := updated.(Model)
	if pm.active {
		t.Error("expected prompt to deactivate after no")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from no")
	}
	msg := cmd().(PromptResultMsg)
	if msg.Confirmed {
		t.Error("expected confirmed=false for no")
	}
}

func TestPrompt_ConfirmNo_Uppercase(t *testing.T) {
	m := NewConfirmModel("Run full sync?")
	_, cmd := m.Update(tea.KeyPressMsg{Code: NKey})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from N key")
	}
	msg := cmd().(PromptResultMsg)
	if msg.Confirmed {
		t.Error("expected confirmed=false for N")
	}
}

func TestPrompt_ConfirmEsc(t *testing.T) {
	m := NewConfirmModel("Run full sync?")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: escapeKey})
	pm := updated.(Model)
	if pm.active {
		t.Error("expected prompt to deactivate after Esc")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Esc cancel")
	}
	msg := cmd().(PromptResultMsg)
	if msg.Confirmed {
		t.Error("expected confirmed=false for Esc cancel")
	}
}

func TestPrompt_ConfirmIgnoresUnknownKey(t *testing.T) {
	m := NewConfirmModel("Run full sync?")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'x'})
	pm := updated.(Model)
	if !pm.active {
		t.Error("expected prompt to stay active on unknown key")
	}
	if cmd != nil {
		t.Error("expected nil cmd on unknown key")
	}
}

func TestPrompt_CategorySelect_InitialCursorAtMatched(t *testing.T) {
	cats := []string{"dev-tools", "frameworks", "libraries", "cli"}
	m := NewCategorySelectModel("Select category", cats, "frameworks")
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 for 'frameworks', got %d", m.cursor)
	}
}

func TestPrompt_CategorySelect_InitialCursorNoMatch(t *testing.T) {
	cats := []string{"dev-tools", "frameworks"}
	m := NewCategorySelectModel("Select category", cats, "nonexistent")
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 for no match, got %d", m.cursor)
	}
}

func TestPrompt_CategorySelect_JK(t *testing.T) {
	cats := []string{"dev-tools", "frameworks", "libraries", "cli"}
	m := NewCategorySelectModel("Select category", cats, "")
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: jKey})
	pm := updated.(Model)
	if pm.cursor != 1 {
		t.Errorf("expected cursor 1 after j, got %d", pm.cursor)
	}
	updated2, _ := pm.Update(tea.KeyPressMsg{Code: kKey})
	pm2 := updated2.(Model)
	if pm2.cursor != 0 {
		t.Errorf("expected cursor 0 after k, got %d", pm2.cursor)
	}
}

func TestPrompt_CategorySelect_Arrows(t *testing.T) {
	cats := []string{"dev-tools", "frameworks", "libraries", "cli"}
	m := NewCategorySelectModel("Select category", cats, "")
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	pm := updated.(Model)
	if pm.cursor != 1 {
		t.Errorf("expected cursor 1 after Down, got %d", pm.cursor)
	}
	updated2, _ := pm.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	pm2 := updated2.(Model)
	if pm2.cursor != 0 {
		t.Errorf("expected cursor 0 after Up, got %d", pm2.cursor)
	}
}

func TestPrompt_CategorySelect_ClampTop(t *testing.T) {
	cats := []string{"dev-tools", "frameworks"}
	m := NewCategorySelectModel("Select category", cats, "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: kKey})
	pm := updated.(Model)
	if pm.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", pm.cursor)
	}
}

func TestPrompt_CategorySelect_ClampTopArrow(t *testing.T) {
	cats := []string{"dev-tools", "frameworks"}
	m := NewCategorySelectModel("Select category", cats, "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	pm := updated.(Model)
	if pm.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", pm.cursor)
	}
}

func TestPrompt_CategorySelect_ClampBottom(t *testing.T) {
	cats := []string{"dev-tools", "frameworks"}
	m := NewCategorySelectModel("Select category", cats, "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: jKey})
	updated2, _ := updated.(Model).Update(tea.KeyPressMsg{Code: jKey})
	pm := updated2.(Model)
	if pm.cursor != 1 {
		t.Errorf("expected cursor clamped at 1, got %d", pm.cursor)
	}
}

func TestPrompt_CategorySelect_ClampBottomArrow(t *testing.T) {
	cats := []string{"dev-tools", "frameworks"}
	m := NewCategorySelectModel("Select category", cats, "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	updated2, _ := updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown})
	pm := updated2.(Model)
	if pm.cursor != 1 {
		t.Errorf("expected cursor clamped at 1, got %d", pm.cursor)
	}
}

func TestPrompt_CategorySelect_EnterReturnsValue(t *testing.T) {
	cats := []string{"dev-tools", "frameworks", "libraries"}
	m := NewCategorySelectModel("Select category", cats, "")
	_, cmd := m.Update(tea.KeyPressMsg{Code: enterKey})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	msg := cmd().(PromptResultMsg)
	if !msg.Confirmed {
		t.Error("expected confirmed=true")
	}
	if msg.Value != "dev-tools" {
		t.Errorf("expected 'dev-tools', got '%s'", msg.Value)
	}
}

func TestPrompt_CategorySelect_EscCancels(t *testing.T) {
	cats := []string{"dev-tools", "frameworks"}
	m := NewCategorySelectModel("Select category", cats, "")
	_, cmd := m.Update(tea.KeyPressMsg{Code: escapeKey})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Esc")
	}
	msg := cmd().(PromptResultMsg)
	if msg.Confirmed {
		t.Error("expected confirmed=false for Esc")
	}
}

func TestPrompt_TagEdit_Typing(t *testing.T) {
	m := NewTagEditModel("Edit tags", "+awesome")
	updated, _ := m.Update(tea.KeyPressMsg{Code: '-'})
	pm := updated.(Model)
	updated2, _ := pm.Update(tea.KeyPressMsg{Code: 'o'})
	pm2 := updated2.(Model)
	updated3, _ := pm2.Update(tea.KeyPressMsg{Code: 'l'})
	pm3 := updated3.(Model)
	updated4, _ := pm3.Update(tea.KeyPressMsg{Code: 'd'})
	pm4 := updated4.(Model)
	if pm4.input != "+awesome-old" {
		t.Errorf("expected '+awesome-old', got '%s'", pm4.input)
	}
}

func TestPrompt_TagEdit_Backspace(t *testing.T) {
	m := NewTagEditModel("Edit tags", "+tui")
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	pm := updated.(Model)
	if pm.input != "+tu" {
		t.Errorf("expected '+tu', got '%s'", pm.input)
	}
}

func TestPrompt_TagEdit_BackspaceOnEmpty(t *testing.T) {
	m := NewTagEditModel("Edit tags", "")
	updated, _ := m.Update(tea.KeyPressMsg{Code: backspaceKey})
	pm := updated.(Model)
	if pm.input != "" {
		t.Errorf("expected empty input, got '%s'", pm.input)
	}
}

func TestPrompt_TagEdit_EnterReturnsTrimmedValue(t *testing.T) {
	m := NewTagEditModel("Edit tags", "  +tui  ")
	_, cmd := m.Update(tea.KeyPressMsg{Code: enterKey})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	msg := cmd().(PromptResultMsg)
	if !msg.Confirmed {
		t.Error("expected confirmed=true")
	}
	if msg.Value != "+tui" {
		t.Errorf("expected '+tui' (trimmed), got '%s'", msg.Value)
	}
}

func TestPrompt_TagEdit_EscCancels(t *testing.T) {
	m := NewTagEditModel("Edit tags", "+tui")
	_, cmd := m.Update(tea.KeyPressMsg{Code: escapeKey})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Esc")
	}
	msg := cmd().(PromptResultMsg)
	if msg.Confirmed {
		t.Error("expected confirmed=false for Esc")
	}
}

func TestPrompt_IsFocused(t *testing.T) {
	m := NewConfirmModel("Run?")
	if !m.IsFocused() {
		t.Error("expected IsFocused=true initially")
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: escapeKey})
	pm := updated.(Model)
	if pm.IsFocused() {
		t.Error("expected IsFocused=false after Esc")
	}
}

func TestPrompt_View_EmptyWhenInactive(t *testing.T) {
	m := NewConfirmModel("Run?")
	updated, _ := m.Update(tea.KeyPressMsg{Code: escapeKey})
	pm := updated.(Model)
	got := pm.View().Content
	if got != "" {
		t.Errorf("expected empty view when inactive, got '%s'", got)
	}
}

func TestPrompt_View_Golden(t *testing.T) {
	tests := []struct {
		name  string
		model Model
	}{
		{"confirm", NewConfirmModel("Run full sync? This may take a while")},
		{
			"category_select",
			func() Model {
				m := NewCategorySelectModel("Select category", []string{"dev-tools", "frameworks", "libraries"}, "frameworks")
				m.width = 80
				return m
			}(),
		},
		{"tag_edit", NewTagEditModel("Edit tags (e.g. +awesome,-old)", "+tui")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.model.View().Content)

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

func TestPrompt_View_Golden_Narrow(t *testing.T) {
	tests := []struct {
		name  string
		model Model
	}{
		{"confirm_narrow", func() Model {
			m := NewConfirmModel("Run full sync?")
			m.width = 40
			return m
		}()},
		{"category_select_narrow", func() Model {
			m := NewCategorySelectModel("Select category", []string{"dev-tools", "frameworks", "libraries"}, "frameworks")
			m.width = 40
			return m
		}()},
		{"tag_edit_narrow", func() Model {
			m := NewTagEditModel("Edit tags", "+tui")
			m.width = 40
			return m
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.model.View().Content)

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
