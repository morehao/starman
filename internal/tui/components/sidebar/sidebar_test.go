package sidebar

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/theme"
)

func TestSidebarViewEmptyShowsNothingSelected(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme: theme.DefaultTheme(),
	}
	m := NewModel(ctx)
	out := m.View()
	if !strings.Contains(out, "Nothing selected...") {
		t.Fatalf("missing empty message: %q", out)
	}
}

func TestSidebarViewRendersContent(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme: theme.DefaultTheme(),
	}
	m := NewModel(ctx)
	m.SetSize(40, 20)
	m.SetContent("details")
	out := m.View()
	if !strings.Contains(out, "details") {
		t.Fatalf("missing content: %q", out)
	}
}
