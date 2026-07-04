package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/starssection"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

func lines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func TestFooterAlwaysVisibleInNormalMode(t *testing.T) {
	sizes := []struct {
		w, h int
	}{
		{120, 40},
		{80, 30},
		{80, 24},
		{50, 20},
	}

	for _, sz := range sizes {
		t.Run(fmt.Sprintf("%dx%d", sz.w, sz.h), func(t *testing.T) {
			ctx := tuicontext.NewContext(config.Default(), nil, "test")
			m := NewModel(ctx)
			updated, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
			m = updated.(Model)

			repos := make([]*store.Repository, 50)
			for i := 0; i < 50; i++ {
				repos[i] = &store.Repository{
					FullName:        fmt.Sprintf("owner/repo-%03d", i),
					StargazersCount: 100 + i,
					Language:        "Go",
				}
			}
			updated, _ = m.Update(starssection.ReposFetchedMsg{SectionID: 1, Repos: repos})
			m = updated.(Model)

			out := m.View().Content
			total := lines(out)
			lastLine := strings.Split(out, "\n")[total-1]
			// On narrow screens, the footer may wrap; check for marker text
			hasFooter := strings.Contains(lastLine, "j/k move") ||
				strings.Contains(lastLine, "1/50") ||
				(total > 1 && strings.Contains(strings.Split(out, "\n")[total-2], "j/k move"))

			if total > sz.h && sz.w >= 80 {
				t.Errorf("total=%d lines, exceeds ScreenHeight=%d (diff=%+d)", total, sz.h, total-sz.h)
			}
			if !hasFooter {
				t.Errorf("footer not found near last line: %q", lastLine)
			}
		})
	}
}
