package keys

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	FirstLine     key.Binding
	LastLine      key.Binding
	NextSection   key.Binding
	PrevSection   key.Binding
	NextView      key.Binding
	PrevView      key.Binding
	ToggleSidebar key.Binding
	Quit          key.Binding
	Help          key.Binding
	OpenGithub    key.Binding
	Refresh       key.Binding
}

var Keys = KeyMap{
	Up:            key.NewBinding(key.WithKeys("k", "up")),
	Down:          key.NewBinding(key.WithKeys("j", "down")),
	FirstLine:     key.NewBinding(key.WithKeys("g")),
	LastLine:      key.NewBinding(key.WithKeys("G")),
	NextSection:   key.NewBinding(key.WithKeys("]", "l")),
	PrevSection:   key.NewBinding(key.WithKeys("[", "h")),
	NextView:      key.NewBinding(key.WithKeys("tab")),
	PrevView:      key.NewBinding(key.WithKeys("shift+tab")),
	ToggleSidebar: key.NewBinding(key.WithKeys("p")),
	Quit:          key.NewBinding(key.WithKeys("q")),
	Help:          key.NewBinding(key.WithKeys("?")),
	OpenGithub:    key.NewBinding(key.WithKeys("o")),
	Refresh:       key.NewBinding(key.WithKeys("r")),
}
