package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/app"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type SyncModel struct {
	store      store.Store
	theme      *styles.Theme
	syncAction app.SyncAction
	enqueue    func(label string) string
	status     string
	progress   int
	total      int
	width      int
	height     int
	syncing    bool
	fullSync   bool
}

type syncDoneMsg struct {
	id     string
	result *app.SyncResult
	err    error
}

func NewSync(s store.Store, theme *styles.Theme, syncAction app.SyncAction, enqueue func(string) string) *SyncModel {
	return &SyncModel{store: s, theme: theme, syncAction: syncAction, enqueue: enqueue, status: "Press Enter to start sync"}
}

func (m *SyncModel) Init() tea.Cmd { return nil }

func (m *SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		if msg.String() == "enter" && !m.syncing {
			m.syncing = true
			m.status = "Syncing..."
			m.total = 0
			m.progress = 0
			if m.syncAction == nil {
				m.syncing = false
				m.status = "Error: sync not configured"
				return m, nil
			}
			id := "sync-task"
			if m.enqueue != nil {
				id = m.enqueue("sync")
			}
			return m, tea.Batch(
				func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "Sync"} },
				m.runSyncCmd(id),
			)
		}
	case syncDoneMsg:
		m.syncing = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
		} else if msg.result != nil {
			m.status = fmt.Sprintf("Done. %d repos synced.", msg.result.Fetched)
		} else {
			m.status = "Done."
		}
		return m, func() tea.Msg { return types.TaskDoneMsg{ID: msg.id, Err: msg.err} }
	}
	return m, nil
}

func (m *SyncModel) runSyncCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := m.syncAction.Run(ctx, app.SyncOpts{Full: m.fullSync})
		return syncDoneMsg{id: id, result: result, err: err}
	}
}

func (m *SyncModel) View() string {
	title := m.theme.PageTitle.Render("Sync") + "\n\n"
	bar := ""
	if m.syncing && m.total > 0 {
		pct := float64(m.progress) / float64(m.total)
		barLen := 40
		filled := int(pct * float64(barLen))
		bar = "[" + strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barLen-filled) + "] "
		bar += fmt.Sprintf("%.0f%%\n", pct*100)
	}
	info := fmt.Sprintf("Status: %s\n", m.status)
	hint := m.theme.HelpText.Render("\nEnter: start sync  Esc: back")
	return title + bar + info + hint
}
