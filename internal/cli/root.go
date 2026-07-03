package cli

import (
	"fmt"
	"os"

	"github.com/morehao/starman/internal/tui"
	"github.com/spf13/cobra"
)

func NewRootCmd(ver string) *cobra.Command {
	root := &cobra.Command{
		Use:     "starman",
		Short:   "Manage your GitHub stars with AI",
		Long:    "starman syncs your GitHub stars, analyzes them with AI, and generates awesome lists.",
		Version: ver,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			return tui.RunTUI(cfg)
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.PersistentFlags().String("config", "", "config file path (default ~/.starman/config.yaml)")
	root.PersistentFlags().String("token", "", "GitHub token (overrides config/env)")
	root.PersistentFlags().Bool("verbose", false, "verbose output")

	root.AddCommand(newSyncCmd())
	root.AddCommand(newGenerateCmd())
	root.AddCommand(newAnalyzeCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newReleaseCmd())
	root.AddCommand(newStarCmd())
	root.AddCommand(newUnstarCmd())
	root.AddCommand(newBackupCmd())
	root.AddCommand(newCategorizeCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newCompletionCmd(root))
	root.AddCommand(newStatsCmd())
	root.AddCommand(newInfoCmd())
	root.AddCommand(newTagCmd())
	root.AddCommand(newTrendingCmd())
	return root
}

func Run(ver string) {
	root := NewRootCmd(ver)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
