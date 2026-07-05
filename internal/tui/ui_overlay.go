package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type visibleChar struct {
	char rune
	ansi string
}

func extractVisibleChars(s string) []visibleChar {
	var chars []visibleChar
	var currentANSI strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			currentANSI.WriteRune(r)
			continue
		}
		if inEscape {
			currentANSI.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		chars = append(chars, visibleChar{char: r, ansi: currentANSI.String()})
		currentANSI.Reset()
	}
	return chars
}

func overlayOnBackground(bg string, dialog string, screenW, screenH int, dimBg color.Color) string {
	bgLines := strings.Split(bg, "\n")
	dialogLines := strings.Split(dialog, "\n")

	dH := len(dialogLines)
	dW := 0
	for _, l := range dialogLines {
		if w := lipgloss.Width(l); w > dW {
			dW = w
		}
	}

	startY := (screenH - dH) / 2
	startX := (screenW - dW) / 2

	dimStyle := lipgloss.NewStyle().Faint(true).Background(dimBg)
	spaceStyle := lipgloss.NewStyle().Background(dimBg)
	dialogBgStyle := lipgloss.NewStyle().Background(dimBg)

	result := make([]string, 0, screenH)
	for y := 0; y < screenH && y < len(bgLines); y++ {
		overlaps := y >= startY && y < startY+dH
		if overlaps {
			oIdx := y - startY
			if oIdx < len(dialogLines) {
				dialogLine := dialogLines[oIdx]
				dLineW := lipgloss.Width(dialogLine)

				dialogWithBg := dialogBgStyle.Render(dialogLine)

				leftPart := ""
				if startX > 0 {
					truncated := truncateVisible(bgLines[y], startX)
					leftPart = dimStyle.Render(truncated)
					leftW := lipgloss.Width(leftPart)
					if leftW < startX {
						leftPart += spaceStyle.Width(startX - leftW).Render("")
					}
				}

				rightW := screenW - startX - dLineW
				rightPart := ""
				if rightW > 0 {
					rightPart = spaceStyle.Width(rightW).Render("")
				}

				middlePart := mergeVisibleChars(
					dimStyle.Render(sliceVisible(bgLines[y], startX, dLineW)),
					dialogWithBg,
				)

				line := leftPart + middlePart + rightPart
				lineW := lipgloss.Width(line)
				if lineW < screenW {
					line += spaceStyle.Width(screenW - lineW).Render("")
				}
				result = append(result, line)
				continue
			}
		}

		dimmed := dimStyle.Render(bgLines[y])
		dimmedW := lipgloss.Width(dimmed)
		if dimmedW < screenW {
			dimmed += spaceStyle.Width(screenW - dimmedW).Render("")
		}
		result = append(result, dimmed)
	}

	return strings.Join(result, "\n")
}

func truncateVisible(s string, max int) string {
	if max <= 0 {
		return ""
	}
	var out strings.Builder
	var buf strings.Builder
	count := 0
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			buf.WriteRune(r)
			continue
		}
		if inEscape {
			buf.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		if count < max {
			out.WriteRune(r)
			count++
		} else {
			out.WriteString(buf.String())
			buf.Reset()
		}
	}
	out.WriteString(buf.String())
	return out.String()
}

func sliceVisible(s string, start, length int) string {
	if length <= 0 {
		return ""
	}
	var out strings.Builder
	var buf strings.Builder
	count := 0
	inEscape := false
	started := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			buf.WriteRune(r)
			continue
		}
		if inEscape {
			buf.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		if count >= start && count < start+length {
			if !started {
				out.WriteString(buf.String())
				buf.Reset()
				started = true
			}
			out.WriteRune(r)
		} else {
			buf.Reset()
		}
		count++
		if count >= start+length {
			break
		}
	}
	return out.String()
}

func mergeVisibleChars(bgSegment, dialogSegment string) string {
	bgChars := extractVisibleChars(bgSegment)
	dlgChars := extractVisibleChars(dialogSegment)

	maxLen := len(dlgChars)
	if len(bgChars) < maxLen {
		maxLen = len(bgChars)
	}

	var out strings.Builder
	for i := 0; i < maxLen; i++ {
		if dlgChars[i].char != ' ' {
			out.WriteString(dlgChars[i].ansi)
			out.WriteRune(dlgChars[i].char)
		} else {
			out.WriteString(bgChars[i].ansi)
			out.WriteRune(bgChars[i].char)
		}
	}
	for i := maxLen; i < len(bgChars); i++ {
		out.WriteString(bgChars[i].ansi)
		out.WriteRune(bgChars[i].char)
	}
	for i := maxLen; i < len(dlgChars); i++ {
		out.WriteString(dlgChars[i].ansi)
		out.WriteRune(dlgChars[i].char)
	}

	return out.String()
}
