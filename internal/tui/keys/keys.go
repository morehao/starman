package keys

import (
	"charm.land/bubbles/v2/key"

	"github.com/morehao/starman/internal/config"
)

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	FirstLine     key.Binding
	LastLine      key.Binding
	NextSection   key.Binding
	PrevSection   key.Binding
	NextGroup     key.Binding
	PrevGroup     key.Binding
	NextView      key.Binding
	PrevView      key.Binding
	ToggleSidebar key.Binding
	Quit          key.Binding
	Help          key.Binding
	OpenGithub    key.Binding
	Refresh       key.Binding
	Sync          key.Binding
	Search        key.Binding
	Command       key.Binding
	Escape        key.Binding
	Enter         key.Binding
	ToggleStar    key.Binding
	EditCategory  key.Binding
	EditTag       key.Binding
	Analyze       key.Binding
	ActionsMenu   key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:            key.NewBinding(key.WithKeys("k", "up")),
		Down:          key.NewBinding(key.WithKeys("j", "down")),
		FirstLine:     key.NewBinding(key.WithKeys("g")),
		LastLine:      key.NewBinding(key.WithKeys("G")),
		NextSection:   key.NewBinding(key.WithKeys("l")),
		PrevSection:   key.NewBinding(key.WithKeys("h")),
		NextGroup:     key.NewBinding(key.WithKeys("]")),
		PrevGroup:     key.NewBinding(key.WithKeys("[")),
		NextView:      key.NewBinding(key.WithKeys("tab")),
		PrevView:      key.NewBinding(key.WithKeys("shift+tab")),
		ToggleSidebar: key.NewBinding(key.WithKeys("p")),
		Quit:          key.NewBinding(key.WithKeys("q")),
		Help:          key.NewBinding(key.WithKeys("?")),
		OpenGithub:    key.NewBinding(key.WithKeys("o")),
		Refresh:       key.NewBinding(key.WithKeys("r")),
		Sync:          key.NewBinding(key.WithKeys("s")),
		Search:        key.NewBinding(key.WithKeys("/")),
		Command:       key.NewBinding(key.WithKeys(":")),
		Escape:        key.NewBinding(key.WithKeys("esc")),
		Enter:         key.NewBinding(key.WithKeys("enter")),
		ToggleStar:    key.NewBinding(key.WithKeys("x")),
		EditCategory:  key.NewBinding(key.WithKeys("c")),
		EditTag:       key.NewBinding(key.WithKeys("t")),
		Analyze:       key.NewBinding(key.WithKeys("a")),
		ActionsMenu:   key.NewBinding(key.WithKeys("o")),
	}
}

var Keys = DefaultKeyMap()

func NewKeyMap(cfg *config.TUIKeybindings) KeyMap {
	km := DefaultKeyMap()
	if cfg == nil {
		return km
	}
	applyBindingOverride(&km.Quit, cfg.Universal.Quit)
	applyBindingOverride(&km.Refresh, cfg.Universal.Refresh)
	applyBindingOverride(&km.Search, cfg.Universal.Search)
	applyBindingOverride(&km.Command, cfg.Universal.Command)
	applyBindingOverride(&km.ToggleSidebar, cfg.Universal.ToggleSidebar)
	applyBindingOverride(&km.Help, cfg.Universal.Help)
	applyBindingOverride(&km.Sync, cfg.Stars.Sync)
	applyBindingOverride(&km.Analyze, cfg.Stars.Analyze)
	applyBindingOverride(&km.ToggleStar, cfg.Stars.ToggleStar)
	applyBindingOverride(&km.EditCategory, cfg.Stars.EditCat)
	applyBindingOverride(&km.EditTag, cfg.Stars.EditTag)
	return km
}

func applyBindingOverride(target *key.Binding, override string) {
	if override != "" {
		*target = key.NewBinding(key.WithKeys(override))
	}
}
