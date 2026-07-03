package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

var ErrNoConfig = fmt.Errorf("config not found")

func Run(cfg *config.Config, ver string) error {
	if cfg == nil {
		return ErrNoConfig
	}
	return runProgram(cfg, ver)
}

func runProgram(cfg *config.Config, ver string) error {
	storeDir, err := config.DefaultDir()
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}

	db, err := store.Open(storeDir + "/starman.db")
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	ctx := tuicontext.NewContext(cfg, db, ver)
	program := tea.NewProgram(NewModel(ctx))
	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}
