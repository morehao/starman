package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/starssection"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

func keyPress(text string) tea.KeyMsg {
	var code rune
	if text != "" {
		code = []rune(text)[0]
	}
	return tea.KeyPressMsg{Text: text, Code: code}
}

func initModel(m Model) Model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return updated.(Model)
}

func TestUIEmptyState(t *testing.T) {
	ctx := tuicontext.NewContext(config.Default(), nil, "test")
	m := initModel(NewModel(ctx))
	out := m.View().Content
	if !strings.Contains(out, "No repos yet") {
		t.Fatalf("want empty state, got: %q", out)
	}
}

func TestUIKeyBindingsFirstLastToggleSidebarAndHelp(t *testing.T) {
	ctx := tuicontext.NewContext(config.Default(), nil, "test")
	m := initModel(NewModel(ctx))
	updated, _ := m.Update(starssection.ReposFetchedMsg{SectionID: 1, Repos: []*store.Repository{
		{FullName: "owner/repo1"},
		{FullName: "owner/repo2"},
	}})
	m = updated.(Model)

	out := m.View().Content
	if !strings.Contains(out, "owner/repo1") {
		t.Fatalf("sidebar should be visible by default, got: %q", out)
	}

	updated, _ = m.Update(keyPress("?"))
	m = updated.(Model)
	out = m.View().Content
	if !strings.Contains(out, "g/G first/last") {
		t.Fatalf("help should show key hints, got: %q", out)
	}

	updated, _ = m.Update(keyPress("p"))
	m = updated.(Model)
	out = m.View().Content
	if !strings.Contains(out, "owner/repo1") {
		t.Fatalf("p should toggle preview showing owner/repo1, got: %q", out)
	}

	updated, _ = m.Update(keyPress("g"))
	m = updated.(Model)
	if m.stars.CurrRowIndex() != 0 {
		t.Fatalf("g should jump to first row, cursor=%d", m.stars.CurrRowIndex())
	}

	updated, _ = m.Update(keyPress("G"))
	m = updated.(Model)
	if m.stars.CurrRowIndex() != 1 {
		t.Fatalf("G should jump to last row, cursor=%d", m.stars.CurrRowIndex())
	}

	updated, _ = m.Update(keyPress("g"))
	m = updated.(Model)
	if m.stars.CurrRowIndex() != 0 {
		t.Fatalf("g should jump to first row, cursor=%d", m.stars.CurrRowIndex())
	}

}
