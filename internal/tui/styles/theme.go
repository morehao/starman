package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Muted      lipgloss.Color
	Background lipgloss.Color
	Text       lipgloss.Color
	Subtle     lipgloss.Color

	Sidebar       lipgloss.Style
	SidebarActive lipgloss.Style
	StatusBar     lipgloss.Style
	Card          lipgloss.Style
	CardTitle     lipgloss.Style
	TableHeader   lipgloss.Style
	TableCell     lipgloss.Style
	Input         lipgloss.Style
	PageTitle     lipgloss.Style
	HelpText      lipgloss.Style
}

func DefaultTheme() *Theme {
	t := &Theme{
		Primary:    lipgloss.Color("#7C3AED"),
		Secondary:  lipgloss.Color("#3B82F6"),
		Success:    lipgloss.Color("#10B981"),
		Warning:    lipgloss.Color("#F59E0B"),
		Error:      lipgloss.Color("#EF4444"),
		Muted:      lipgloss.Color("#6B7280"),
		Background: lipgloss.Color("#1F2937"),
		Text:       lipgloss.Color("#F9FAFB"),
		Subtle:     lipgloss.Color("#9CA3AF"),
	}

	t.Sidebar = lipgloss.NewStyle().
		Width(24).
		Height(100).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(t.Muted).
		Padding(1, 1)

	t.SidebarActive = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Primary).
		Bold(true)

	t.StatusBar = lipgloss.NewStyle().
		Height(1).
		Background(t.Muted).
		Foreground(t.Text).
		Padding(0, 1)

	t.Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Padding(1)

	t.CardTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary)

	t.TableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Padding(0, 1)

	t.TableCell = lipgloss.NewStyle().
		Padding(0, 1)

	t.Input = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Secondary).
		Padding(0, 1)

	t.PageTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Padding(0, 1).
		MarginBottom(1)

	t.HelpText = lipgloss.NewStyle().
		Foreground(t.Subtle)

	return t
}
