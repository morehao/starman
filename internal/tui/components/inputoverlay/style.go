package inputoverlay

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

const OverlayWidth = 42

func RenderOverlay(th theme.Theme, width, height int, title, query string) string {
	dialogWidth := OverlayWidth
	if width > 0 && width < dialogWidth+4 {
		dialogWidth = width - 4
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.FaintBorder).
		Padding(1, 2).
		Width(dialogWidth)

	titleStyle := lipgloss.NewStyle().
		Foreground(th.PrimaryText).
		Bold(true)

	inputLabelStyle := lipgloss.NewStyle().
		Foreground(th.FaintText)

	inputStyle := lipgloss.NewStyle().
		Foreground(th.PrimaryText)

	hintStyle := lipgloss.NewStyle().
		Foreground(th.FaintText)

	contentWidth := dialogWidth - 6
	if contentWidth < 0 {
		contentWidth = 0
	}
	separator := ""
	if contentWidth > 0 {
		separator = lipgloss.NewStyle().
			Foreground(th.FaintBorder).
			Render(strings.Repeat("─", contentWidth))
	}

	queryDisplay := query
	if queryDisplay == "" {
		queryDisplay = " "
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')
	if separator != "" {
		b.WriteString(separator)
		b.WriteByte('\n')
	}
	b.WriteString(inputLabelStyle.Render("🔍 "))
	b.WriteString(inputStyle.Render(queryDisplay + "█"))
	b.WriteByte('\n')
	b.WriteString(hintStyle.Render("Enter to confirm  Esc to cancel"))

	rendered := dialogStyle.Render(b.String())

	if width == 0 || height == 0 {
		return rendered
	}

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		rendered,
	)
}
