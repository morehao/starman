package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/morehao/starman/internal/config"
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


