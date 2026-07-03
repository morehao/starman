package listviewport

import "testing"

func TestCursorNavigation(t *testing.T) {
	m := New(nil)
	m.SetRows([]Row{{Columns: []string{"a"}}, {Columns: []string{"b"}}})
	if m.Cursor() != 0 {
		t.Fatalf("cursor=%d want=0", m.Cursor())
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
