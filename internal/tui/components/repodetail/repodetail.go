package repodetail

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"

	"github.com/morehao/starman/internal/store"
)

type Model struct {
	repo  *store.Repository
	width int
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) SetRepo(r *store.Repository) { m.repo = r }

func (m *Model) SetWidth(w int) { m.width = w }

func (m Model) Repo() *store.Repository { return m.repo }

func (m Model) View() string {
	if m.repo == nil {
		return ""
	}
	return m.renderDetail()
}

func (m Model) renderDetail() string {
	var b strings.Builder

	repoName := lipgloss.NewStyle().Bold(true).Render(m.repo.FullName)
	meta := fmt.Sprintf("  \u2B50%d  \U0001F374%d", m.repo.StargazersCount, m.repo.ForksCount)
	b.WriteString(repoName + meta)

	if m.repo.Language != "" {
		b.WriteString("  " + m.repo.Language)
	}
	b.WriteString("\n")

	b.WriteString(strings.Repeat("\u2500", 50))
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
			lock = " \U0001F512"
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
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("\u25B6 AI Summary"))
		b.WriteString("\n")
		b.WriteString(m.renderMarkdown(m.repo.AISummary))
	}

	if m.repo.Homepage != "" {
		b.WriteString(fmt.Sprintf("\n\n%s  %s", fieldLabel("Homepage"), m.repo.Homepage))
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("\u2500", 50))
	b.WriteString("\n")

	if m.repo.SubscribedReleases {
		b.WriteString(fmt.Sprintf("Release: Subscribed  |  Last fetched: %v", m.repo.LastReleaseFetch))
	} else {
		b.WriteString("Release: Not subscribed")
	}

	return b.String()
}

func (m Model) renderMarkdown(content string) string {
	renderWidth := m.width - 4
	if renderWidth < 20 {
		renderWidth = 20
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return rendered
}
