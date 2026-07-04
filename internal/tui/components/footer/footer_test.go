package footer

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/theme"
)

func TestFooterViewContainsHelpHint(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:     theme.DefaultTheme(),
		View:      context.StarsView,
	}
	f := NewModel(ctx)
	out := f.View()
	if !strings.Contains(out, "j/k move") {
		t.Fatalf("missing help text: %q", out)
	}
}

func TestFooterViewNoViewSwitcher(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:     theme.DefaultTheme(),
		View:      context.StarsView,
	}
	f := NewModel(ctx)
	out := f.View()
	if strings.Contains(out, "Stars") {
		t.Fatalf("unexpected Stars in footer: %q", out)
	}
	if strings.Contains(out, "Trending") {
		t.Fatalf("unexpected Trending in footer: %q", out)
	}
}

func TestFooterViewShowsPagerWhenSet(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:     theme.DefaultTheme(),
		View:      context.StarsView,
	}
	f := NewModel(ctx)
	f.SetPager("3/248")
	out := f.View()
	if !strings.Contains(out, "3/248") {
		t.Fatalf("missing pager: %q", out)
	}
}
