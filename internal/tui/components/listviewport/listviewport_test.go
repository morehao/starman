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

func TestFitColumns(t *testing.T) {
	tests := []struct {
		name      string
		columns   []Column
		availW    int
		wantWidths []int
		wantUsed  int
	}{
		{
			name:       "single flex column gets extra space",
			columns:    []Column{{Title: "Name", Width: 10, Flex: true}},
			availW:     30,
			wantWidths: []int{28},
			wantUsed:   30,
		},
		{
			name: "multiple flex columns only first gets extra",
			columns: []Column{
				{Title: "A", Width: 5, Flex: true},
				{Title: "B", Width: 5, Flex: true},
			},
			availW:     30,
			wantWidths: []int{22, 5},
			wantUsed:   30,
		},
		{
			name: "no flex columns no expansion",
			columns: []Column{
				{Title: "A", Width: 10},
				{Title: "B", Width: 15},
			},
			availW:     50,
			wantWidths: []int{10, 15},
			wantUsed:   28,
		},
		{
			name:       "zero extra space exact fit",
			columns:    []Column{{Title: "Name", Width: 10, Flex: true}},
			availW:     12,
			wantWidths: []int{10},
			wantUsed:   12,
		},
		{
			name:       "no columns at all",
			columns:    nil,
			availW:     100,
			wantWidths: nil,
			wantUsed:   2,
		},
		{
			name: "flex mixed with non-flex in middle position",
			columns: []Column{
				{Title: "A", Width: 6},
				{Title: "B", Width: 6, Flex: true},
				{Title: "C", Width: 6},
			},
			availW:     40,
			wantWidths: []int{6, 24, 6},
			wantUsed:   40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{columns: tt.columns}
			gotCols, gotUsed := m.fitColumns(tt.availW)

			if len(gotCols) != len(tt.wantWidths) {
				t.Fatalf("got %d columns, want %d", len(gotCols), len(tt.wantWidths))
			}
			for i, wantW := range tt.wantWidths {
				if gotCols[i].Width != wantW {
					t.Errorf("column[%d].Width = %d, want %d", i, gotCols[i].Width, wantW)
				}
			}
			if gotUsed != tt.wantUsed {
				t.Errorf("used = %d, want %d", gotUsed, tt.wantUsed)
			}
		})
	}
}
