package repoview

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

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

	var b strings.Builder

	b.WriteString(m.renderOverview())

	readme := m.renderReadme()
	if readme != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("README"))
		b.WriteString("\n")
		b.WriteString(readme)
	}

	releases := m.renderReleases()
	if releases != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Releases"))
		b.WriteString("\n")
		b.WriteString(releases)
	}

	return b.String()
}

func (m Model) renderOverview() string {
	if m.repo == nil {
		return ""
	}

	wrapW := m.wrapWidth()

	var b strings.Builder

	repoName := lipgloss.NewStyle().Hyperlink("https://github.com/"+m.repo.FullName).Render(m.repo.FullName)
	meta := fmt.Sprintf("  ⭐%d  🍴%d", m.repo.StargazersCount, m.repo.ForksCount)
	b.WriteString(repoName + meta)

	if m.repo.Language != "" {
		b.WriteString("  " + m.repo.Language)
	}
	b.WriteString("\n")

	sepW := wrapW
	if sepW <= 0 {
		sepW = 50
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.NoColor{}).Render(strings.Repeat("─", sepW)))
	b.WriteString("\n")

	fieldLabel := func(s string) string {
		return lipgloss.NewStyle().Bold(true).Width(14).Render(s)
	}

	// wrapField wraps a field value to fit after the label, indenting continuation lines.
	wrapField := func(label, value string) string {
		if wrapW <= 0 {
			return value
		}
		labelStr := fieldLabel(label)
		labelW := lipgloss.Width(labelStr)
		valW := wrapW - labelW - 2
		if valW < 10 {
			valW = wrapW
		}
		value = wordWrap(value, valW)
		if strings.Contains(value, "\n") {
			indent := strings.Repeat(" ", labelW+2)
			value = strings.ReplaceAll(value, "\n", "\n"+indent)
		}
		return value
	}

	if m.repo.Description != "" {
		desc := m.repo.Description
		if wrapW > 0 {
			desc = wordWrap(desc, wrapW)
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Description"))
		b.WriteString("\n")
		b.WriteString(desc)
		b.WriteString("\n")
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

	if m.repo.URL != "" {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("URL"), wrapField("URL", m.repo.URL)))
	}

	if m.repo.StarredAt != "" {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Starred"), m.repo.StarredAt))
	}

	if m.repo.RepoUpdatedAt != "" {
		updated := m.repo.RepoUpdatedAt
		if t, err := time.Parse(time.RFC3339, updated); err == nil {
			updated = t.Format("2006-01-02 15:04")
		}
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Updated"), updated))
	}

	if m.repo.AnalyzedAt != nil {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Analyzed"), m.repo.AnalyzedAt.Format("2006-01-02 15:04")))
	}

	if len(m.repo.Topics) > 0 {
		topics := strings.Join(m.repo.Topics, ", ")
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Topics"), wrapField("Topics", topics)))
	}

	allTags := append([]string{}, m.repo.AITags...)
	allTags = append(allTags, m.repo.CustomTags...)
	if len(allTags) > 0 {
		tags := strings.Join(allTags, ", ")
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Tags"), wrapField("Tags", tags)))
	}

	if m.repo.AISummary != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("AI Summary"))
		b.WriteString("\n")
		summary := m.repo.AISummary
		if wrapW > 0 {
			summary = wordWrap(summary, wrapW)
		}
		b.WriteString(summary)
	}

	if m.repo.Homepage != "" {
		homepage := wrapField("Homepage", m.repo.Homepage)
		homepage = lipgloss.NewStyle().Hyperlink(m.repo.Homepage).Render(homepage)
		b.WriteString(fmt.Sprintf("\n\n%s  %s", fieldLabel("Homepage"), homepage))
	}

	return b.String()
}

func (m Model) renderReadme() string {
	if m.repo == nil {
		return ""
	}
	wrapW := m.wrapWidth()
	if wrapW > 0 {
		return wordWrap(m.repo.AISummary, wrapW)
	}
	return m.repo.AISummary
}

func (m Model) renderReleases() string {
	if m.repo == nil {
		return ""
	}

	if m.repo.SubscribedReleases {
		return fmt.Sprintf("Subscribed to releases for %s\n\nLast fetched: %v", m.repo.FullName, m.repo.LastReleaseFetch)
	}
	return "Not subscribed to releases."
}

func (m Model) wrapWidth() int {
	if m.width <= 0 {
		return 0
	}
	w := m.width - 1
	if w < 20 {
		w = m.width
	}
	return w
}

// wordWrap wraps text to fit within the given width, preserving existing newlines.
func wordWrap(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}
	var result strings.Builder
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if lipgloss.Width(line) <= width {
			result.WriteString(line)
			continue
		}
		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			test := current + " " + word
			if lipgloss.Width(test) > width {
				result.WriteString(current)
				result.WriteString("\n")
				current = word
			} else {
				current = test
			}
		}
		result.WriteString(current)
	}
	return result.String()
}
