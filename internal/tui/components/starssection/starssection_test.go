package starssection

import (
	"context"
	"testing"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

type mockStore struct {
	repos []*store.Repository
	store.Store
}

func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	return m.repos, nil
}

func TestStarsSectionFetchAndNavigate(t *testing.T) {
	ctx := &tuicontext.ProgramContext{Store: &mockStore{repos: []*store.Repository{
		{FullName: "owner/repo1", URL: "https://github.com/owner/repo1"},
		{FullName: "owner/repo2", URL: "https://github.com/owner/repo2"},
	}}}

	s := NewModel(1, ctx, section.SectionConfig{Title: "All"}, GroupAll)
	cmds := s.FetchNextPageSectionRows()
	if len(cmds) == 0 {
		t.Fatalf("expected fetch cmd")
	}

	updated, _ := s.Update(cmds[0]())
	if updated.NumRows() != 2 {
		t.Fatalf("rows=%d want=2", updated.NumRows())
	}
	if updated.CurrRow().GetTitle() != "owner/repo1" {
		t.Fatalf("title=%q want=owner/repo1", updated.CurrRow().GetTitle())
	}

	updated.NextRow()
	if updated.CurrRow().GetTitle() != "owner/repo2" {
		t.Fatalf("title=%q want=owner/repo2", updated.CurrRow().GetTitle())
	}
}
