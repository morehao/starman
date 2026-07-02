package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type AnalyzeModel struct {
	store    store.Store
	theme    *styles.Theme
	status   string
	analyzed int
	failed   int
	total    int
	width    int
	height   int
	running  bool
}

func NewAnalyze(s store.Store, theme *styles.Theme) *AnalyzeModel {
	return &AnalyzeModel{store: s, theme: theme, status: "Press Enter to start analysis"}
}

func (m *AnalyzeModel) Init() tea.Cmd { return nil }

func (m *AnalyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		if msg.String() == "enter" && !m.running {
			m.running = true
			m.status = "Analyzing..."
			m.total = 10
			return m, m.stepAnalyzeCmd
		}
	case analyzeStepMsg:
		m.analyzed = msg.done
		m.failed += msg.failed
		if msg.done >= m.total {
			m.running = false
			m.status = fmt.Sprintf("Done. Analyzed %d, failed %d.", m.analyzed, m.failed)
		} else {
			return m, m.stepAnalyzeCmd
		}
	}
	return m, nil
}

type analyzeStepMsg struct{ done, failed int }

func (m *AnalyzeModel) stepAnalyzeCmd() tea.Msg {
	return analyzeStepMsg{done: m.analyzed + 1, failed: 0}
}

func (m *AnalyzeModel) View() string {
	title := m.theme.PageTitle.Render("Analyze") + "\n\n"
	info := fmt.Sprintf("Progress: %d/%d\nStatus: %s\nFailed: %d\n",
		m.analyzed, m.total, m.status, m.failed)
	hint := m.theme.HelpText.Render("\nEnter: start  Esc: back")
	return title + info + hint
}
