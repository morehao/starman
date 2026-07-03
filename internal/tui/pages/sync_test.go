package pages

import (
	"context"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/app"
	"github.com/morehao/starman/internal/tui/styles"
)

type mockSyncAction struct {
	result *app.SyncResult
	err    error
}

func (m *mockSyncAction) Run(ctx context.Context, opts app.SyncOpts) (*app.SyncResult, error) {
	return m.result, m.err
}

func TestSyncPageEnterWithRealAction(t *testing.T) {
	mock := &mockSyncAction{result: &app.SyncResult{Fetched: 42}}
	m := NewSync(nil, styles.DefaultTheme(), mock, func(label string) string { return "task-1" })
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil command on Enter")
	}
	if !m.syncing {
		t.Fatalf("expected syncing state after Enter")
	}
}

func TestSyncPageNoSyncAction(t *testing.T) {
	m := NewSync(nil, styles.DefaultTheme(), nil, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.syncing {
		t.Fatalf("expected syncing to be reset when syncAction is nil")
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd when syncAction is nil")
	}
	if m.status != "Error: sync not configured" {
		t.Fatalf("expected error status, got %q", m.status)
	}
}

func TestSyncPageInit(t *testing.T) {
	m := NewSync(nil, styles.DefaultTheme(), nil, nil)
	cmd := m.Init()
	if cmd != nil {
		t.Fatalf("expected nil init command, got non-nil")
	}
}

func TestSyncPageInitialStatus(t *testing.T) {
	m := NewSync(nil, styles.DefaultTheme(), nil, nil)
	if m.status != "Press Enter to start sync" {
		t.Fatalf("unexpected initial status: %s", m.status)
	}
}

func TestSyncPageViewNonEmpty(t *testing.T) {
	m := NewSync(nil, styles.DefaultTheme(), nil, nil)
	v := m.View()
	if v == "" {
		t.Fatalf("expected non-empty view")
	}
}
