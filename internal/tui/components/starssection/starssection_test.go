package starssection

import (
	"context"
	"errors"
	"testing"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

type mockStore struct {
	repos []*store.Repository
	err   error
	store.Store
}

func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	if m.err != nil {
		return nil, m.err
	}
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

func TestStarsSectionFetchGroupsByLanguageWhenConfigured(t *testing.T) {
	ctx := &tuicontext.ProgramContext{Store: &mockStore{repos: []*store.Repository{
		{FullName: "owner/go1", Language: "Go"},
		{FullName: "owner/zig", Language: "Zig"},
		{FullName: "owner/go2", Language: "Go"},
	}}}

	s := NewModel(1, ctx, section.SectionConfig{Title: "By language"}, "language")
	cmds := s.FetchNextPageSectionRows()
	if len(cmds) != 1 {
		t.Fatalf("expected one fetch cmd, got %d", len(cmds))
	}

	updated, _ := s.Update(cmds[0]())
	if updated.NumRows() != 5 {
		t.Fatalf("rows=%d want=5 (2 group headers + 3 repos)", updated.NumRows())
	}

	row0 := updated.CurrRow()
	if row0.GetTitle() != "Go (2)" {
		t.Fatalf("first row=%q want=Go (2)", row0.GetTitle())
	}
	if _, ok := row0.(GroupHeaderRow); !ok {
		t.Fatalf("first row should be group header, got %T", row0)
	}

	updated.NextRow()
	if updated.CurrRow().GetTitle() != "owner/go1" {
		t.Fatalf("second row=%q want=owner/go1", updated.CurrRow().GetTitle())
	}
	updated.LastItem()
	if updated.CurrRow().GetTitle() != "owner/zig" {
		t.Fatalf("last row=%q want=owner/zig", updated.CurrRow().GetTitle())
	}
}

func TestStarsSectionFetchReturnsErrorMessageOnStoreFailure(t *testing.T) {
	ctx := &tuicontext.ProgramContext{Store: &mockStore{err: errors.New("boom")}}
	s := NewModel(1, ctx, section.SectionConfig{Title: "All"}, GroupAll)
	cmds := s.FetchNextPageSectionRows()
	if len(cmds) != 1 {
		t.Fatalf("expected one fetch cmd, got %d", len(cmds))
	}

	msg := cmds[0]()
	failed, ok := msg.(ReposFetchFailedMsg)
	if !ok {
		t.Fatalf("expected ReposFetchFailedMsg, got %T", msg)
	}
	if failed.SectionID != 1 {
		t.Fatalf("section id=%d want=1", failed.SectionID)
	}
	if failed.Err == nil || failed.Err.Error() != "boom" {
		t.Fatalf("error=%v want boom", failed.Err)
	}

	updated, _ := s.Update(failed)
	if updated.GetIsLoading() {
		t.Fatalf("isLoading should be false after failure")
	}
	if updated.NumRows() != 0 {
		t.Fatalf("rows=%d want=0", updated.NumRows())
	}
}
