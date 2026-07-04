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

func TestFooterVisibleAfterManyKeyPresses(t *testing.T) {
	for _, h := range []int{40, 30, 24} {
		t.Run(fmt.Sprintf("h=%d", h), func(t *testing.T) {
			ctx := tuicontext.NewContext(config.Default(), nil, "test")
			m := NewModel(ctx)
			updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: h})
			m = updated.(Model)

			// Load 200 repos to have a long list
			repos := make([]*store.Repository, 200)
			for i := 0; i < 200; i++ {
				repos[i] = &store.Repository{
					FullName:        fmt.Sprintf("owner/repo-%03d", i),
					StargazersCount: 100 + i,
					Language:        "Go",
				}
			}
			updated, _ = m.Update(starssection.ReposFetchedMsg{SectionID: 1, Repos: repos})
			m = updated.(Model)

			// Navigate down 50 times
			for i := 0; i < 50; i++ {
				updated, _ = m.Update(keyPress("j"))
				m = updated.(Model)

				out := m.View().Content
				total := lines(out)
				ls := strings.Split(out, "\n")
				hasFooter := strings.Contains(ls[total-1], "Stars")

				if total != h {
					t.Errorf("j press %d: total=%d expected=%d", i, total, h)
				}
				if !hasFooter {
					t.Errorf("j press %d: footer missing on last line: %q", i, ls[total-1])
				}
			}

			// Navigate back up 50 times
			for i := 0; i < 50; i++ {
				updated, _ = m.Update(keyPress("k"))
				m = updated.(Model)

				out := m.View().Content
				total := lines(out)
				ls := strings.Split(out, "\n")
				hasFooter := strings.Contains(ls[total-1], "Stars")

				if total != h {
					t.Errorf("k press %d: total=%d expected=%d", i, total, h)
				}
				if !hasFooter {
					t.Errorf("k press %d: footer missing on last line: %q", i, ls[total-1])
				}
			}
		})
	}
}

// Test with spinner tick active (simulating ongoing task)
func TestFooterVisibleWithActiveSpinner(t *testing.T) {
	ctx := tuicontext.NewContext(config.Default(), nil, "test")
	m := NewModel(ctx)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
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

	// Simulate an active task (spinner will tick)
	updated, _ = m.Update(TaskStartedMsg{TaskID: "test-1", Name: "testing..."})
	m = updated.(Model)

	// Send multiple spinner ticks and navigation interleaved
	for i := 0; i < 20; i++ {
		// spinner tick
		updated, _ = m.Update(spinnerTickMsg{frame: i})
		m = updated.(Model)

		// navigation
		updated, _ = m.Update(keyPress("j"))
		m = updated.(Model)

		out := m.View().Content
		total := lines(out)
		ls := strings.Split(out, "\n")
		hasFooter := strings.Contains(ls[total-1], "Stars")

		if total != 40 {
			t.Errorf("step %d: total=%d expected=40", i, total)
		}
		if !hasFooter {
			t.Errorf("step %d: footer missing on last line: %q", i, ls[total-1])
		}
	}
}

// Test with various repos having different data lengths (long descriptions, etc.)
func TestFooterVisibleWithDiverseRepos(t *testing.T) {
	ctx := tuicontext.NewContext(config.Default(), nil, "test")
	m := NewModel(ctx)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	repos := make([]*store.Repository, 10)
	for i := 0; i < 10; i++ {
		desc := ""
		if i == 3 {
			desc = strings.Repeat("A very long description that might cause rendering issues in the sidebar. ", 10)
		}
		repos[i] = &store.Repository{
			FullName:        fmt.Sprintf("owner/repo-%03d", i),
			StargazersCount: 100 + i,
			Language:        "Go",
			Description:     desc,
			AICategory:      "ai-tools",
			AIPlatforms:     []string{"web", "cli"},
			Topics:          []string{"go", "cli", "tool"},
			AITags:          []string{"productivity", "dev"},
			CustomTags:      []string{"favorite"},
			AISummary:       strings.Repeat("This is a test summary. ", 5),
			StarredAt:       "2024-01-15",
		}
	}
	updated, _ = m.Update(starssection.ReposFetchedMsg{SectionID: 1, Repos: repos})
	m = updated.(Model)

	for i := 0; i < 10; i++ {
		updated, _ = m.Update(keyPress("j"))
		m = updated.(Model)

		out := m.View().Content
		total := lines(out)
		ls := strings.Split(out, "\n")
		hasFooter := strings.Contains(ls[total-1], "Stars")

		if total != 40 {
			t.Errorf("j press %d: total=%d expected=40, repo=%s", i, total, repos[i].FullName)
		}
		if !hasFooter {
			t.Errorf("j press %d: footer missing: %q", i, ls[total-1])
		}
	}
}

// Test switching sidebar tabs during navigation  
func TestFooterVisibleWhenSwitchingSidebarTabs(t *testing.T) {
	ctx := tuicontext.NewContext(config.Default(), nil, "test")
	m := NewModel(ctx)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)

	repos := make([]*store.Repository, 10)
	for i := 0; i < 10; i++ {
		repos[i] = &store.Repository{
			FullName:        fmt.Sprintf("owner/repo-%03d", i),
			StargazersCount: 100 + i,
			Language:        "Go",
			AISummary:       fmt.Sprintf("# Summary %d\n\nThis is repo %d.", i, i),
		}
	}
	updated, _ = m.Update(starssection.ReposFetchedMsg{SectionID: 1, Repos: repos})
	m = updated.(Model)

	// Switch to README tab (tab 1)
	for _, key := range []string{"l", "j", "j", "l", "j"} {
		updated, _ = m.Update(keyPress(key))
		m = updated.(Model)

		out := m.View().Content
		total := lines(out)
		ls := strings.Split(out, "\n")
		hasFooter := strings.Contains(ls[total-1], "Stars")

		if total != 40 {
			t.Errorf("after '%s': total=%d expected=40", key, total)
		}
		if !hasFooter {
			t.Errorf("after '%s': footer missing: %q", key, ls[total-1])
		}
	}
}
