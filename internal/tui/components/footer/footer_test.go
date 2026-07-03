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
	if !strings.Contains(out, "?help") {
		t.Fatalf("missing help hint: %q", out)
	}
}

func TestFooterViewContainsViewSwitcher(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:     theme.DefaultTheme(),
		View:      context.StarsView,
	}
	f := NewModel(ctx)
	out := f.View()
	if !strings.Contains(out, "Stars") {
		t.Fatalf("missing Stars view: %q", out)
	}
	if !strings.Contains(out, "Trending") {
		t.Fatalf("missing Trending view: %q", out)
	}
	if !strings.Contains(out, "Releases") {
		t.Fatalf("missing Releases view: %q", out)
	}
	if !strings.Contains(out, "Stats") {
		t.Fatalf("missing Stats view: %q", out)
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
