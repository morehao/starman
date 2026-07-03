package context

import (
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/common"
	"github.com/morehao/starman/internal/tui/theme"
)

type ViewType string

const (
	StarsView    ViewType = "stars"
	TrendingView ViewType = "trending"
	ReleasesView ViewType = "releases"
	StatsView    ViewType = "stats"
)

type ProgramContext struct {
	Config               *config.Config
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
}

func NewContext(cfg *config.Config, s store.Store, ver string) *ProgramContext {
	t := theme.DefaultTheme()
	return &ProgramContext{
		Config:          cfg,
		Store:           s,
		Version:         ver,
		View:            StarsView,
		SidebarOpen:     true,
		PreviewPosition: "right",
		Theme:           t,
		Styles:          common.BuildStyles(t),
	}
}
