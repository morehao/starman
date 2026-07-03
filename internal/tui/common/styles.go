package common

import (
	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/constants"
	"github.com/morehao/starman/internal/tui/theme"
)

type CommonStyles struct {
	MainTextStyle  lipgloss.Style
	FaintTextStyle lipgloss.Style
	FooterStyle    lipgloss.Style
	ErrorStyle     lipgloss.Style
	FailureGlyph   string
	SuccessGlyph   string
}

func BuildStyles(t theme.Theme) CommonStyles {
	return CommonStyles{
		MainTextStyle:  lipgloss.NewStyle().Foreground(t.PrimaryText).Bold(true),
		FaintTextStyle: lipgloss.NewStyle().Foreground(t.FaintText),
		FooterStyle:    lipgloss.NewStyle().Background(t.SelectedBackground).Height(constants.FooterHeight),
		ErrorStyle:     lipgloss.NewStyle().Foreground(t.ErrorText),
		FailureGlyph:   lipgloss.NewStyle().Foreground(t.ErrorText).Render(constants.FailureIcon),
		SuccessGlyph:   lipgloss.NewStyle().Foreground(t.SuccessText).Render(constants.SuccessIcon),
	}
}

func RenderPreviewHeader(t theme.Theme, width int, text string) string {
	return lipgloss.NewStyle().
		PaddingLeft(1).
		Width(width).
		Background(t.SelectedBackground).
		Foreground(t.SecondaryText).
		Render(text)
}
