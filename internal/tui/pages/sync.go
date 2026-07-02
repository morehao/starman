package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type SyncModel struct {
	store    store.Store
	theme    *styles.Theme
	status   string
	progress int
	total    int
	width    int
	height   int
	syncing  bool
}

func NewSync(s store.Store, theme *styles.Theme) *SyncModel {
	return &SyncModel{store: s, theme: theme, status: "Press Enter to start sync"}
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
			m.total = 10
			return m, m.stepSyncCmd
		}
	case syncStepMsg:
		m.progress = msg.step
		m.status = fmt.Sprintf("Fetched page %d/%d", msg.step, m.total)
		if msg.step >= m.total {
			m.syncing = false
			m.status = fmt.Sprintf("Done. %d new repos found.", msg.step*2)
		} else {
			return m, m.stepSyncCmd
		}
	}
	return m, nil
}

type syncStepMsg struct{ step int }

func (m *SyncModel) stepSyncCmd() tea.Msg {
	_ = context.Background()
	return syncStepMsg{step: m.progress + 1}
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
