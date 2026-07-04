package repoview

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"

	"github.com/morehao/starman/internal/store"
)

var tabTitles = []string{"Overview", "README", "Releases"}

type Model struct {
	activeTab int
	repo      *store.Repository
	width     int
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) SetRepo(r *store.Repository) { m.repo = r }

func (m *Model) SetWidth(w int) { m.width = w }

func (m Model) Repo() *store.Repository { return m.repo }

func (m *Model) NextTab() { m.activeTab = (m.activeTab + 1) % len(tabTitles) }

func (m *Model) PrevTab() { m.activeTab = (m.activeTab + len(tabTitles) - 1) % len(tabTitles) }

func (m Model) ActiveTab() int { return m.activeTab }

func (m Model) View() string {
	var parts []string

	tabLine := m.renderTabs()
	parts = append(parts, tabLine)

	if m.repo == nil {
		return strings.Join(parts, "\n")
	}

	switch m.activeTab {
	case 0:
		parts = append(parts, m.renderOverview())
	case 1:
		parts = append(parts, m.renderReadme())
	case 2:
		parts = append(parts, m.renderReleases())
	}
	return strings.Join(parts, "\n")
}

func (m Model) renderTabs() string {
	tabParts := make([]string, 0, len(tabTitles))
	for i, tab := range tabTitles {
		if i == m.activeTab {
			tabParts = append(tabParts, "[ "+tab+" ]")
		} else {
			tabParts = append(tabParts, "  "+tab+"  ")
		}
	}
	return strings.Join(tabParts, "│")
}

func (m Model) renderOverview() string {
	if m.repo == nil {
		return ""
	}

	var b strings.Builder

	repoName := lipgloss.NewStyle().Bold(true).Render(m.repo.FullName)
	meta := fmt.Sprintf("  ⭐%d  🍴%d", m.repo.StargazersCount, m.repo.ForksCount)
	b.WriteString(repoName + meta)

	if m.repo.Language != "" {
		b.WriteString("  " + m.repo.Language)
	}
	b.WriteString("\n")

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.NoColor{}).Render(strings.Repeat("─", 50)))
	b.WriteString("\n")

	if m.repo.Description != "" {
		b.WriteString("\n" + m.repo.Description)
	}

	fieldLabel := func(s string) string {
		return lipgloss.NewStyle().Bold(true).Width(10).Render(s)
	}

	if m.repo.Language != "" {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Language"), m.repo.Language))
	}

	if m.repo.AICategory != "" || m.repo.CustomCategory != "" {
		cat := m.repo.AICategory
		if cat == "" {
			cat = m.repo.CustomCategory
		}
		lock := ""
		if m.repo.CategoryLocked {
			lock = " 🔒"
		}
		b.WriteString(fmt.Sprintf("\n%s  %s%s", fieldLabel("Category"), cat, lock))
	}

	if len(m.repo.AIPlatforms) > 0 {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Platform"), strings.Join(m.repo.AIPlatforms, ", ")))
	}

	if m.repo.StarredAt != "" {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Starred"), m.repo.StarredAt))
	}

	if len(m.repo.Topics) > 0 {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Topics"), strings.Join(m.repo.Topics, ", ")))
	}

	allTags := append([]string{}, m.repo.AITags...)
	allTags = append(allTags, m.repo.CustomTags...)
	if len(allTags) > 0 {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Tags"), strings.Join(allTags, ", ")))
	}

	if m.repo.AISummary != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("AI Summary"))
		b.WriteString("\n")
		b.WriteString(m.repo.AISummary)
	}

	if m.repo.Homepage != "" {
		b.WriteString(fmt.Sprintf("\n\n%s  %s", fieldLabel("Homepage"), m.repo.Homepage))
	}

	return b.String()
}

func (m Model) renderReadme() string {
	if m.repo == nil {
		return ""
	}

	readmeContent := m.repo.AISummary
	if readmeContent == "" {
		return "No README available.\n\nUse :analyze to generate an AI summary."
	}

	renderWidth := m.width - 4
	if renderWidth < 20 {
		renderWidth = 20
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return "AI Summary\n\n" + readmeContent
	}

	rendered, err := renderer.Render(readmeContent)
	if err != nil {
		return "AI Summary\n\n" + readmeContent
	}
	return rendered
}

func (m Model) renderReleases() string {
	if m.repo == nil {
		return ""
	}

	if m.repo.SubscribedReleases {
		return fmt.Sprintf("Subscribed to releases for %s\n\nLast fetched: %v", m.repo.FullName, m.repo.LastReleaseFetch)
	}
	return "Not subscribed to releases.\n\nPress 's' to subscribe."
}
