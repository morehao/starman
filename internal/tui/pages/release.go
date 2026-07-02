package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type releaseTab int

const (
	tabSubscribed releaseTab = iota
	tabUnread
)

type ReleaseModel struct {
	store      store.Store
	theme      *styles.Theme
	tab        releaseTab
	subscribed []*store.Repository
	cursor     int
	width      int
	height     int
	loaded     bool
}

func NewRelease(s store.Store, theme *styles.Theme) *ReleaseModel {
	return &ReleaseModel{store: s, theme: theme}
}

func (m *ReleaseModel) Init() tea.Cmd { return m.loadCmd }

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

func (m *ReleaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case reposLoadedMsg:
		m.subscribed = msg.repos
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.tab == tabSubscribed {
				m.tab = tabUnread
			} else {
				m.tab = tabSubscribed
			}
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			limit := len(m.subscribed)
			m.cursor = min(m.cursor+1, limit-1)
		}
	}
	return m, nil
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
	items := m.subscribed
	if m.tab == tabSubscribed {
		for i, r := range items {
			line := fmt.Sprintf("  %-40s subscribed", truncate(r.FullName, 38))
			if i == m.cursor {
				body += m.theme.SidebarActive.Render(line) + "\n"
			} else {
				body += line + "\n"
			}
		}
	} else {
		body += "Loading unread releases...\n"
	}
	return title + body
}
