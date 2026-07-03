package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	SelectedBackground color.Color
	PrimaryText        color.Color
	SecondaryText      color.Color
	FaintText          color.Color
	SuccessText        color.Color
	ErrorText          color.Color
	WarningText        color.Color
	FaintBorder        color.Color
}

func DefaultTheme() Theme {
	return Theme{
		SelectedBackground: lipgloss.Color("#1E1E2E"),
		PrimaryText:        lipgloss.Color("#CDD6F4"),
		SecondaryText:      lipgloss.Color("#A6ADC8"),
		FaintText:          lipgloss.Color("#6C7086"),
		SuccessText:        lipgloss.Color("#A6E3A1"),
		ErrorText:          lipgloss.Color("#F38BA8"),
		WarningText:        lipgloss.Color("#F9E2AF"),
		FaintBorder:        lipgloss.Color("#313244"),
	}
}
