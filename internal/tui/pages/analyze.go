package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type AnalyzeModel struct {
	store      store.Store
	theme      *styles.Theme
	analyzer   *ai.BatchAnalyzer
	enqueue    func(string) string
	status     string
	analyzed   int
	failed     int
	total      int
	width      int
	height     int
	running    bool
	taskID     string
	progressCh chan analyzeProgressMsg
}

type analyzeProgressMsg struct {
	id     string
	done   int
	total  int
	failed int
	final  bool
}

func NewAnalyze(s store.Store, theme *styles.Theme, analyzer *ai.BatchAnalyzer, enqueue func(string) string) *AnalyzeModel {
	return &AnalyzeModel{store: s, theme: theme, analyzer: analyzer, enqueue: enqueue, status: "Press Enter to start analysis"}
}

func (m *AnalyzeModel) Init() tea.Cmd { return nil }

func (m *AnalyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		if msg.String() == "enter" && !m.running {
			if m.analyzer == nil {
				m.status = "Error: AI analyzer not configured"
				return m, nil
			}
			m.running = true
			m.status = "Analyzing..."
			m.total = 0
			m.analyzed = 0
			m.failed = 0
			id := m.enqueue("analyze")
			m.taskID = id
			return m, tea.Batch(
				func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "Analyze"} },
				m.runAnalyzeCmd(id),
			)
		}
	case analyzeProgressMsg:
		if msg.final {
			m.running = false
			m.status = fmt.Sprintf("Done. Analyzed %d, failed %d.", msg.done, msg.failed)
			return m, func() tea.Msg { return types.TaskDoneMsg{ID: msg.id, Err: nil} }
		}
		m.analyzed = msg.done
		m.total = msg.total
		m.failed = msg.failed
		return m, m.waitForProgress()
	}
	return m, nil
}

func (m *AnalyzeModel) runAnalyzeCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		repos, err := m.store.ListRepositories(ctx)
		if err != nil {
			return analyzeProgressMsg{id: id, final: true, done: 0, failed: 0}
		}
		m.progressCh = make(chan analyzeProgressMsg, len(repos)+1)
		go func() {
			result, err := m.analyzer.Run(ctx, repos, ai.BatchOpts{
				OnProgress: func(done, total int, _ string) {
					m.progressCh <- analyzeProgressMsg{id: id, done: done, total: total}
				},
			})
			var failed int
			if result != nil {
				failed = result.Failed
			}
			if err != nil && result != nil {
				failed = result.Total
			}
			done := 0
			if result != nil {
				done = result.Total
			}
			m.progressCh <- analyzeProgressMsg{id: id, done: done, failed: failed, final: true}
			close(m.progressCh)
		}()
		msg, ok := <-m.progressCh
		if !ok {
			return analyzeProgressMsg{id: id, final: true}
		}
		return msg
	}
}

func (m *AnalyzeModel) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.progressCh
		if !ok {
			return analyzeProgressMsg{final: true}
		}
		return msg
	}
}

func (m *AnalyzeModel) View() string {
	title := m.theme.PageTitle.Render("Analyze") + "\n\n"
	info := fmt.Sprintf("Progress: %d/%d\nStatus: %s\nFailed: %d\n",
		m.analyzed, m.total, m.status, m.failed)
	hint := m.theme.HelpText.Render("\nEnter: start  Esc: back")
	return title + info + hint
}
