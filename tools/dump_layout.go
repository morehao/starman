package main

import (
	"fmt"
	"strings"

	"github.com/morehao/starman/internal/tui/components"
	"github.com/morehao/starman/internal/tui/styles"
)

func main() {
	theme := styles.DefaultTheme()
	fmt.Println("== workspace (Sync with params + output, 80 cols) ==")
	w := components.NewCommandWorkspace(theme)
	w.SelectCommand(components.CommandNode{
		ID:          "sync",
		Label:       "Sync Repositories",
		Description: "拉取 GitHub 星标并合并到本地库",
		Shortcut:    "s",
		Params: []components.CommandParamSpec{
			{Key: "include_archived", Label: "include archived", Type: components.ParamBool, Required: false, DefaultValue: "false"},
			{Key: "concurrency", Label: "concurrency", Type: components.ParamNumber, Required: false, DefaultValue: "3"},
			{Key: "target", Label: "target", Type: components.ParamText, Required: false, DefaultValue: "all"},
		},
	})
	w.AppendOutput("fetching starred repos... 340 found")
	w.AppendOutput("✓ anthropic/claude-code")
	w.AppendOutput("✓ facebook/react")
	w.AppendOutput("✗ some/private-repo (403 forbidden)")
	w.AppendOutput("synced 128 repos, updated 17")
	fmt.Println(w.View())

	fmt.Println("\n== workspace (no command selected) ==")
	w2 := components.NewCommandWorkspace(theme)
	fmt.Println(w2.View())

	fmt.Println("\n== workspace (no params, no output) ==")
	w3 := components.NewCommandWorkspace(theme)
	w3.SelectCommand(components.CommandNode{
		ID:          "dashboard",
		Label:       "Dashboard",
		Description: "总览",
		Shortcut:    "1",
	})
	fmt.Println(w3.View())

	fmt.Println("\n== sidebar (startup, only pinned) ==")
	s := components.NewSidebar(theme)
	fmt.Println(s.View())

	s.SetRecent([]string{"sync", "analyze", "repo-list", "generate"})
	fmt.Println("\n== sidebar (with recent + pinned) ==")
	fmt.Println(s.View())

	fmt.Println("\n== statusbar (140 cols) ==")
	sb := components.NewStatusBar(theme)
	_ = sb
	fmt.Println(strings.Repeat("=", 60))
}
