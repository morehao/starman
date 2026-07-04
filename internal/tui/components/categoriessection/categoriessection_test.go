package categoriessection

import (
	"testing"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

func testStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testContext(t *testing.T, s store.Store) *tuicontext.ProgramContext {
	t.Helper()
	ctx := tuicontext.NewContext(nil, s, "dev")
	return ctx
}

func TestNewModel(t *testing.T) {
	ctx := testContext(t, testStore(t))
	m := NewModel(1, ctx, section.SectionConfig{Title: "Categories"})

	if m.GetId() != 1 {
		t.Errorf("expected id 1, got %d", m.GetId())
	}
	if m.GetType() != "categories" {
		t.Errorf("expected type 'categories', got %s", m.GetType())
	}
}

func TestCategoryRow_GetColumns(t *testing.T) {
	c := &store.Category{
		ID:        "web-app",
		Name:      "Web 应用",
		Keywords:  []string{"web", "framework"},
		SortOrder: 1,
		IsCustom:  false,
	}

	row := CategoryRow{Category: c, RepoCount: 42}
	cols := row.GetColumns()

	if cols[0] != "web-app" {
		t.Errorf("col ID: expected 'web-app', got %s", cols[0])
	}
	if cols[1] != "Web 应用" {
		t.Errorf("col Name: expected 'Web 应用', got %s", cols[1])
	}
	if cols[2] != "web, framework" {
		t.Errorf("col Keywords: expected 'web, framework', got %s", cols[2])
	}
	if cols[3] != "42" {
		t.Errorf("col Repos: expected '42', got %s", cols[3])
	}
	if cols[4] != "1" {
		t.Errorf("col Sort: expected '1', got %s", cols[4])
	}
	if cols[5] != "内置" {
		t.Errorf("col Type: expected '内置', got %s", cols[5])
	}
}

func TestBuildRows(t *testing.T) {
	cats := []*store.Category{
		{ID: "web-app", Name: "Web 应用", SortOrder: 1},
		{ID: "mobile-app", Name: "移动应用", SortOrder: 2},
	}
	repos := []*store.Repository{
		{CustomCategory: "web-app"},
		{AICategory: "web-app"},
		{CustomCategory: "mobile-app"},
	}

	ctx := testContext(t, testStore(t))
	m := NewModel(1, ctx, section.SectionConfig{})
	rows := m.buildRows(cats, repos)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	row0 := rows[0].(CategoryRow)
	if row0.RepoCount != 2 {
		t.Errorf("expected 2 repos for web-app, got %d", row0.RepoCount)
	}
	row1 := rows[1].(CategoryRow)
	if row1.RepoCount != 1 {
		t.Errorf("expected 1 repo for mobile-app, got %d", row1.RepoCount)
	}
}

func TestFetchNextPageSectionRows_NotLoaded(t *testing.T) {
	ctx := testContext(t, testStore(t))
	m := NewModel(1, ctx, section.SectionConfig{})

	cmds := m.FetchNextPageSectionRows()
	if cmds == nil {
		t.Fatal("expected non-nil cmds")
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(cmds))
	}

	msg := cmds[0]()
	fetched, ok := msg.(CategoriesFetchedMsg)
	if !ok {
		t.Fatalf("expected CategoriesFetchedMsg, got %T", msg)
	}
	if fetched.SectionID != 1 {
		t.Errorf("expected sectionID 1, got %d", fetched.SectionID)
	}
}

func TestUpdate_CategoriesFetched(t *testing.T) {
	ctx := testContext(t, testStore(t))
	m := NewModel(1, ctx, section.SectionConfig{})

	cats := []*store.Category{
		{ID: "web-app", Name: "Web 应用", SortOrder: 1},
	}
	msg := CategoriesFetchedMsg{SectionID: 1, Categories: cats, Repos: nil}

	updated, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("Update should not return cmd for CategoriesFetchedMsg")
	}
	nm, ok := updated.(*Model)
	if !ok {
		t.Fatal("expected *Model after Update")
	}
	if !nm.loaded {
		t.Error("expected loaded=true after CategoriesFetchedMsg")
	}
	if nm.isLoading {
		t.Error("expected isLoading=false after CategoriesFetchedMsg")
	}
	if len(nm.rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(nm.rows))
	}
}

func TestCursorBoundaries(t *testing.T) {
	ctx := testContext(t, testStore(t))
	m := NewModel(1, ctx, section.SectionConfig{})
	cats := []*store.Category{
		{ID: "a", Name: "A", SortOrder: 1},
		{ID: "b", Name: "B", SortOrder: 2},
	}
	m.rows = m.buildRows(cats, nil)
	m.list.SetRows(m.rows)
	m.loaded = true

	for i := 0; i < 3; i++ {
		m.PrevRow()
	}
	if m.CurrRowIndex() < 0 {
		t.Error("cursor should not go below 0")
	}

	for i := 0; i < 10; i++ {
		m.NextRow()
	}
	if m.CurrRowIndex() >= len(cats) {
		t.Error("cursor should not exceed row count")
	}
}

func TestFetchNextPageSectionRows_ReturnsNonBlockingCmd(t *testing.T) {
	ctx := testContext(t, testStore(t))
	m := NewModel(1, ctx, section.SectionConfig{})

	cmds := m.FetchNextPageSectionRows()
	if cmds == nil || len(cmds) == 0 {
		t.Fatal("FetchNextPageSectionRows must return non-nil cmds")
	}

	msg := cmds[0]()
	_, ok := msg.(CategoriesFetchedMsg)
	if !ok {
		t.Fatalf("expected CategoriesFetchedMsg, got %T", msg)
	}
}
