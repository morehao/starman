package tui

import (
	"fmt"

	"github.com/morehao/starman/internal/config"
)

var ErrNoConfig = fmt.Errorf("config not found")

func Run(cfg *config.Config, ver string) error {
	if cfg == nil {
		return ErrNoConfig
	}
	return runProgram(cfg, ver)
}

func runProgram(cfg *config.Config, ver string) error {
	_ = cfg
	_ = ver
	return nil
}
