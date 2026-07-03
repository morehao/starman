package sidebar

import (
	"strings"
	"testing"
)

func TestSidebarViewEmptyShowsNothingSelected(t *testing.T) {
	m := NewModel(nil)
	out := m.View()
	if !strings.Contains(out, "Nothing selected...") {
		t.Fatalf("missing empty message: %q", out)
	}
}

func TestSidebarViewRendersContent(t *testing.T) {
	m := NewModel(nil)
	m.SetContent("details")
	out := m.View()
	if !strings.Contains(out, "details") {
		t.Fatalf("missing content: %q", out)
	}
}
