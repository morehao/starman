package tui

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

var (
	ErrInteractiveRequired = errors.New("interactive required")
	ErrBlockingRequired    = errors.New("blocking required")
)

type CommandRunner interface {
	Run(ctx context.Context, input string) (stdout, stderr string, err error)
}

var Runner CommandRunner

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
	defer func() { _ = db.Close() }()

	ctx := tuicontext.NewContext(cfg, db, ver)
	model := NewModel(ctx)
	model.runner = Runner
	program := tea.NewProgram(model)
	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}
