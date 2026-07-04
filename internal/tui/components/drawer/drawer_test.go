package drawer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
var tsRe = regexp.MustCompile(`\d{2}:\d{2}:\d{2} `)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func stripTimestamp(s string) string {
	return tsRe.ReplaceAllString(s, "")
}

func goldenPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", t.Name()+".golden")
}

func updateGolden() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

func TestDrawer_InitialState(t *testing.T) {
	m := NewModel()
	if m.IsOpen() {
		t.Error("drawer should be closed initially")
	}
	if m.IsFocused() {
		t.Error("drawer should not be focused initially")
	}
}

func TestDrawer_ToggleOpen(t *testing.T) {
	m := NewModel()
	m.SetSize(100, 40)

	msg := tea.KeyPressMsg{Code: ctrlO}
	updated, _ := m.Update(msg)
	dm := updated.(Model)

	if !dm.IsOpen() {
		t.Error("drawer should be open after Ctrl+O")
	}

	msg2 := tea.KeyPressMsg{Code: esc}
	updated2, _ := dm.Update(msg2)
	dm2 := updated2.(Model)

	if dm2.IsOpen() {
		t.Error("drawer should be closed after Escape")
	}
}

func TestDrawer_ClearHistory(t *testing.T) {
	m := NewModel()
	m.AddEntry(":sync", "synced 248 repos", "")
	m.AddEntry(":search go --json", `[{"full_name":"golang/go"}]`, "")

	msg := tea.KeyPressMsg{Code: ctrlL}
	updated, _ := m.Update(msg)
	dm := updated.(Model)

	if len(dm.entries) != 0 {
		t.Errorf("expected 0 entries after Ctrl+L, got %d", len(dm.entries))
	}
}

func TestDrawer_View(t *testing.T) {
	m := NewModel()
	m.SetSize(100, 40)
	m.AddEntry(":sync --full", "248 repos fetched", "")

	view := m.View().Content
	if view != "" {
		t.Error("closed drawer should render empty string")
	}

	msg := tea.KeyPressMsg{Code: ctrlO}
	updated, _ := m.Update(msg)
	dm := updated.(Model)
	view = dm.View().Content
	if view == "" {
		t.Error("open drawer should render content")
	}
	if !strings.Contains(view, ":sync --full") {
		t.Error("drawer view should contain command text")
	}
}

func TestDrawer_View_Golden(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"wide", 120, 40},
		{"standard", 80, 24},
		{"narrow", 50, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()
			m.SetSize(tt.width, tt.height)
			m.AddEntry(":sync --full", "248 repos fetched (12 new) in 3.5s", "")

			msg := tea.KeyPressMsg{Code: ctrlO}
			updated, _ := m.Update(msg)
			dm := updated.(Model)

			got := stripTimestamp(stripANSI(dm.View().Content))
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
