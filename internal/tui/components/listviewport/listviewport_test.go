package listviewport

import (
	"testing"

	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/theme"
)

type fakeRow struct {
	id    string
	title string
	url   string
}

func (r fakeRow) GetId() string      { return r.id }
func (r fakeRow) GetTitle() string   { return r.title }
func (r fakeRow) GetUrl() string     { return r.url }
func (r fakeRow) GetColumns() []string { return []string{r.title} }

func testCtx() *context.ProgramContext {
	return &context.ProgramContext{
		Theme: theme.DefaultTheme(),
	}
}

func testRows(titles ...string) []section.RowData {
	rows := make([]section.RowData, 0, len(titles))
	for _, title := range titles {
		rows = append(rows, fakeRow{id: title, title: title, url: "https://example.com/" + title})
	}
	return rows
}

func TestCursorNavigation(t *testing.T) {
	m := New(nil, testCtx())
	m.SetRows(testRows("a", "b"))
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}
	m.PrevRow()
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}
	m.NextRow()
	if m.Cursor() != 1 {
		t.Fatalf("cursor=%d want=1", m.Cursor())
	}
	m.NextRow()
	if m.Cursor() != 1 {
		t.Fatalf("cursor=%d want=1", m.Cursor())
	}
	m.PrevRow()
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}
}

func TestFirstAndLastItem(t *testing.T) {
	m := New(nil, testCtx())
	m.SetRows(testRows("a", "b", "c"))

	m.LastItem()
	if m.Cursor() != 2 {
		t.Fatalf("cursor=%d want=2", m.Cursor())
	}

	m.FirstItem()
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}

	m.SetRows(nil)
	m.LastItem()
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}
	m.FirstItem()
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}
}

func TestSetRowsEmptyResetsCursorAndPager(t *testing.T) {
	m := New(nil, testCtx())
	m.SetRows(testRows("a", "b"))
	m.LastItem()
	if m.Cursor() != 1 {
		t.Fatalf("cursor=%d want=1", m.Cursor())
	}

	m.SetRows(nil)
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
	}
	if m.Pager() != "0/0" {
		t.Fatalf("pager=%q want=0/0", m.Pager())
	}
}
