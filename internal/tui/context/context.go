package context

import (
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/common"
	"github.com/morehao/starman/internal/tui/keys"
	"github.com/morehao/starman/internal/tui/theme"
)

type ViewType string

const (
	StarsView      ViewType = "stars"
	CategoriesView ViewType = "categories"
	TrendingView   ViewType = "trending"
	ReleasesView ViewType = "releases"
	StatsView    ViewType = "stats"
)

type ProgramContext struct {
	Config               *config.Config
	TUICfg               *config.TUIConfig
	Store                store.Store
	Version              string
	ScreenWidth          int
	ScreenHeight         int
	MainContentWidth     int
	MainContentHeight    int
	Theme                theme.Theme
	Styles               common.CommonStyles
	View                 ViewType
	SidebarOpen          bool
	PreviewPosition      string
	DynamicPreviewWidth  int
	DynamicPreviewHeight int
	Keys                 keys.KeyMap
}

func NewContext(cfg *config.Config, s store.Store, ver string) *ProgramContext {
	t := theme.DefaultTheme()
	if cfg != nil && cfg.TUI.Theme == "light" {
		t = theme.LightTheme()
	}
	ctx := &ProgramContext{
		Config:          cfg,
		Store:           s,
		Version:         ver,
		View:            StarsView,
		SidebarOpen:     true,
		PreviewPosition: "right",
		Theme:           t,
		Styles:          common.BuildStyles(t),
		Keys:            keys.DefaultKeyMap(),
	}

	if cfg != nil {
		ctx.TUICfg = &cfg.TUI
		ctx.SidebarOpen = cfg.TUI.Preview.Open
		ctx.PreviewPosition = cfg.TUI.Preview.Position
		ctx.View = ViewType(cfg.TUI.DefaultView)
		if ctx.View == "" {
			ctx.View = StarsView
		}
		if ctx.PreviewPosition == "" {
			ctx.PreviewPosition = "right"
		}
		ctx.Keys = keys.NewKeyMap(&cfg.TUI.Keybindings)
	}
	return ctx
}
