package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/spf13/cobra"
)

func loadConfig(cmd *cobra.Command) (*config.Config, string, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath, _ = config.DefaultPath()
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, configPath, fmt.Errorf("config not found at %s, run 'starman config init' first", configPath)
		}
		return nil, configPath, err
	}
	return cfg, configPath, nil
}

func openStore() (store.Store, error) {
	dir, err := config.DefaultDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dir, "starman.db")
	return store.Open(dbPath)
}

func resolveGitHubToken(cmd *cobra.Command, cfg *config.Config) string {
	flagToken, _ := cmd.Flags().GetString("token")
	return config.ResolveToken(cfg, flagToken)
}
