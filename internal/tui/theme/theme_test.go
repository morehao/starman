package theme

import (
	"image/color"
	"testing"
)

var noColor color.Color

func TestDefaultTheme_AllTokensSet(t *testing.T) {
	th := DefaultTheme()

	tests := []struct {
		name  string
		value color.Color
	}{
		{"SelectedBackground", th.SelectedBackground},
		{"PrimaryText", th.PrimaryText},
		{"SecondaryText", th.SecondaryText},
		{"FaintText", th.FaintText},
		{"SuccessText", th.SuccessText},
		{"ErrorText", th.ErrorText},
		{"WarningText", th.WarningText},
		{"FaintBorder", th.FaintBorder},
	}
	for _, tt := range tests {
		if tt.value == noColor {
			t.Errorf("DefaultTheme.%s has no color set", tt.name)
		}
	}
}

func TestLightTheme_AllTokensSet(t *testing.T) {
	th := LightTheme()

	tests := []struct {
		name  string
		value color.Color
	}{
		{"SelectedBackground", th.SelectedBackground},
		{"PrimaryText", th.PrimaryText},
		{"SecondaryText", th.SecondaryText},
		{"FaintText", th.FaintText},
		{"SuccessText", th.SuccessText},
		{"ErrorText", th.ErrorText},
		{"WarningText", th.WarningText},
		{"FaintBorder", th.FaintBorder},
	}
	for _, tt := range tests {
		if tt.value == noColor {
			t.Errorf("LightTheme.%s has no color set", tt.name)
		}
	}
}

func TestLightTheme_DifferentFromDark(t *testing.T) {
	dark := DefaultTheme()
	light := LightTheme()

	if dark.PrimaryText == light.PrimaryText {
		t.Error("expected LightTheme PrimaryText to differ from DefaultTheme")
	}
	if dark.SelectedBackground == light.SelectedBackground {
		t.Error("expected LightTheme SelectedBackground to differ from DefaultTheme")
	}
	if dark.FaintText == light.FaintText {
		t.Error("expected LightTheme FaintText to differ from DefaultTheme")
	}
}

func TestDefaultTheme_DarkColors(t *testing.T) {
	th := DefaultTheme()
	lpc, ok := th.PrimaryText.(interface{ RGBA() (uint32, uint32, uint32, uint32) })
	if ok {
		r, g, b, _ := lpc.RGBA()
		brightness := (r + g + b) / 3
		if brightness < 0x8000 {
			t.Error("expected DefaultTheme PrimaryText to be light colored")
		}
	}
}

func TestLightTheme_LightColors(t *testing.T) {
	th := LightTheme()
	lpc, ok := th.SelectedBackground.(interface{ RGBA() (uint32, uint32, uint32, uint32) })
	if ok {
		r, g, b, _ := lpc.RGBA()
		brightness := (r + g + b) / 3
		if brightness < 0x8000 {
			t.Error("expected LightTheme SelectedBackground to be light colored")
		}
	}
}
