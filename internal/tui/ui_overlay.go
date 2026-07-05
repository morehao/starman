package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

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
	spaceStyle := lipgloss.NewStyle().Faint(true).Background(dimBg)

	result := make([]string, 0, screenH)
	for y := 0; y < screenH && y < len(bgLines); y++ {
		overlaps := y >= startY && y < startY+dH
		if overlaps {
			oIdx := y - startY
			if oIdx < len(dialogLines) {
				dialogLine := dialogLines[oIdx]
				dLineW := lipgloss.Width(dialogLine)

				leftPart := ""
				if startX > 0 {
					truncated := truncateVisible(bgLines[y], startX)
					leftPart = dimStyle.Render(truncated)
					leftW := lipgloss.Width(leftPart)
					if leftW < startX {
						leftPart += spaceStyle.Width(startX - leftW).Render("")
					}
				}

				middlePart := dialogLine

				rightW := screenW - startX - dLineW
				rightPart := ""
				if rightW > 0 {
					rightPart = spaceStyle.Width(rightW).Render("")
				}

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
