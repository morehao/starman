package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/release"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type releaseTab int

const (
	tabSubscribed releaseTab = iota
	tabUnread
)

type ReleaseModel struct {
	store      store.Store
	theme      *styles.Theme
	tracker    *release.Tracker
	enqueue    func(string) string
	tab        releaseTab
	subscribed []*store.Repository
	unread     []*store.Release
	cursor     int
	width      int
	height     int
	loaded     bool
	pulling    bool
	taskID     string
}

type releasePulledMsg struct {
	id    string
	stats *release.PullStats
	err   error
}

func NewRelease(s store.Store, theme *styles.Theme, tracker *release.Tracker, enqueue func(string) string) *ReleaseModel {
	return &ReleaseModel{store: s, theme: theme, tracker: tracker, enqueue: enqueue}
}

func (m *ReleaseModel) Init() tea.Cmd {
	return tea.Batch(m.loadCmd, m.pullCmd)
}

func (m *ReleaseModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, err := m.store.ListRepositories(ctx)
	if err != nil {
		return err
	}
	var subs []*store.Repository
	for _, r := range repos {
		if r.SubscribedReleases {
			subs = append(subs, r)
		}
	}
	return reposLoadedMsg{repos: subs}
}

func (m *ReleaseModel) pullCmd() tea.Msg {
	if m.tracker == nil {
		return releasePulledMsg{err: fmt.Errorf("release tracker not configured")}
	}
	id := m.enqueue("release-pull")
	ctx := context.Background()
	stats, err := m.tracker.PullReleases(ctx)
	return releasePulledMsg{id: id, stats: stats, err: err}
}

func (m *ReleaseModel) loadUnreadCmd() tea.Msg {
	ctx := context.Background()
	rels, err := m.store.ListUnreadReleases(ctx)
	if err != nil {
		return err
	}
	return unreadLoadedMsg{releases: rels}
}

type unreadLoadedMsg struct {
	releases []*store.Release
	err      error
}

func (m *ReleaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case reposLoadedMsg:
		m.subscribed = msg.repos
		m.loaded = true
	case unreadLoadedMsg:
		m.unread = msg.releases
	case releasePulledMsg:
		if msg.id != "" {
			return m, tea.Batch(
				func() tea.Msg { return types.TaskDoneMsg{ID: msg.id, Err: msg.err} },
				m.loadUnreadCmd,
			)
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.tab == tabSubscribed {
				m.tab = tabUnread
				if len(m.unread) == 0 {
					return m, m.loadUnreadCmd
				}
			} else {
				m.tab = tabSubscribed
			}
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			limit := m.itemCount()
			m.cursor = min(m.cursor+1, limit-1)
		case "enter":
			if m.tracker == nil {
				return m, nil
			}
			if m.pulling {
				return m, nil
			}
			m.pulling = true
			id := m.enqueue("release-pull")
			m.taskID = id
			return m, tea.Batch(
				func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "Release Pull"} },
				m.pullCmd,
			)
		}
	}
	return m, nil
}

func (m *ReleaseModel) itemCount() int {
	if m.tab == tabSubscribed {
		return len(m.subscribed)
	}
	return len(m.unread)
}

func (m *ReleaseModel) View() string {
	if !m.loaded {
		return "loading releases..."
	}
	title := m.theme.PageTitle.Render("Releases") + "\n"
	tabs := ""
	if m.tab == tabSubscribed {
		tabs += m.theme.SidebarActive.Render("[Subscribed]") + " [Unread] "
	} else {
		tabs += "[Subscribed] " + m.theme.SidebarActive.Render("[Unread]")
	}
	body := tabs + "\n\n"
	if m.tab == tabSubscribed {
		for i, r := range m.subscribed {
			line := fmt.Sprintf("  %-40s subscribed", truncate(r.FullName, 38))
			if i == m.cursor {
				body += m.theme.SidebarActive.Render(line) + "\n"
			} else {
				body += line + "\n"
			}
		}
	} else {
		if len(m.unread) == 0 {
			body += "No unread releases\n"
		} else {
			for i, r := range m.unread {
				line := fmt.Sprintf("  %s %s", truncate(r.RepoFullName, 30), truncate(r.Name, 30))
				if i == m.cursor {
					body += m.theme.SidebarActive.Render(line) + "\n"
				} else {
					body += line + "\n"
				}
			}
		}
	}
	help := m.theme.HelpText.Render("\nTab: switch  Enter: pull releases  Esc: back")
	return title + body + help
}
