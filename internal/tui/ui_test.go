package tui

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/config"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

func TestUIEmptyState(t *testing.T) {
	ctx := tuicontext.NewContext(config.Default(), nil, "test")
	m := NewModel(ctx)
	out := m.View().Content
	if !strings.Contains(out, "No repos yet") {
		t.Fatalf("want empty state, got: %q", out)
	}
}
