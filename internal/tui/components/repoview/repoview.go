package repoview

import (
	"strings"

	"github.com/morehao/starman/internal/store"
)

var tabTitles = []string{"Overview", "README", "Releases"}

type Model struct {
	activeTab int
	repo      *store.Repository
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) SetRepo(r *store.Repository) { m.repo = r }

func (m Model) Repo() *store.Repository { return m.repo }

func (m *Model) NextTab() { m.activeTab = (m.activeTab + 1) % len(tabTitles) }

func (m *Model) PrevTab() { m.activeTab = (m.activeTab + len(tabTitles) - 1) % len(tabTitles) }

func (m Model) ActiveTab() int { return m.activeTab }

func (m Model) View() string {
	parts := make([]string, 0, len(tabTitles))
	for i, tab := range tabTitles {
		if i == m.activeTab {
			parts = append(parts, "["+tab+"]")
		} else {
			parts = append(parts, tab)
		}
	}
	if m.repo == nil {
		return strings.Join(parts, " | ")
	}
	return strings.Join(parts, " | ") + "\n" + m.repo.FullName
}
